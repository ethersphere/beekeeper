package ping

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

var (
	rttGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pingpong",
			Name:      "rtt_duration_seconds",
			Help:      "Ping round-trip time duration Gauge",
		},
		[]string{"node", "peer"},
	)
	rttHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pingpong",
			Name:      "rtt_seconds",
			Help:      "Ping round-trip time duration Histogram",
			Buckets:   prometheus.LinearBuckets(0, 0.003, 10),
		},
	)
)

// Options represents pingpong check options
type Options struct {
	MetricsPusher *push.Pusher
	Seed          int64
}

// compile check whether Ping implements interface
var _ check.Check = (*Ping)(nil)

// Ping check
type Ping struct{}

// NewPing returns new ping check
func NewPing() check.Check {
	return &Ping{}
}

// Run executes ping check
func (p *Ping) Run(ctx context.Context, cluster *bee.Cluster, o interface{}) (err error) {
	fmt.Println("checking pingpong")

	opts, ok := o.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	if err := CheckD(ctx, cluster, opts); err != nil {
		return err
	}

	fmt.Println("pingpong check completed successfully")
	return
}

// CheckD executes ping from all nodes to all other nodes in the cluster
func CheckD(ctx context.Context, cluster *bee.Cluster, o Options) (err error) {
	if o.MetricsPusher != nil {
		o.MetricsPusher.Collector(rttGauge)
		o.MetricsPusher.Collector(rttHistogram)
		o.MetricsPusher.Format(expfmt.FmtText)
	}

	nodeGroups := cluster.NodeGroups()
	for _, ng := range nodeGroups {
		nodesClients, err := ng.NodesClients(ctx)
		if err != nil {
			return fmt.Errorf("get nodes clients: %w", err)
		}

		for n := range nodeStream(ctx, nodesClients) {
			for t := 0; t < 5; t++ {
				time.Sleep(2 * time.Duration(t) * time.Second)

				if n.Error != nil {
					if t == 4 {
						return fmt.Errorf("node %s: %w", n.Name, n.Error)
					}
					fmt.Printf("node %s: %v\n", n.Name, n.Error)
					continue
				}
				fmt.Printf("Node %s: %s Peer: %s RTT: %s\n", n.Name, n.Address, n.PeerAddress, n.RTT)

				rtt, err := time.ParseDuration(n.RTT)
				if err != nil {
					if t == 4 {
						return fmt.Errorf("node %s: %w", n.Name, err)
					}
					fmt.Printf("node %s: %v\n", n.Name, err)
					continue
				}

				rttGauge.WithLabelValues(n.Address.String(), n.PeerAddress.String()).Set(rtt.Seconds())
				rttHistogram.Observe(rtt.Seconds())

				if o.MetricsPusher != nil {
					if err := o.MetricsPusher.Push(); err != nil {
						fmt.Printf("node %s: %v\n", n.Name, err)
					}
				}
				break
			}
		}
	}

	return
}

type nodeStreamMsg struct {
	Name        string
	Address     swarm.Address
	PeerAddress swarm.Address
	RTT         string
	Error       error
}

func nodeStream(ctx context.Context, nodes map[string]*bee.Client) <-chan nodeStreamMsg {
	nodeStream := make(chan nodeStreamMsg)

	var wg sync.WaitGroup
	for k, v := range nodes {
		wg.Add(1)
		go func(name string, node *bee.Client) {
			defer wg.Done()

			address, err := node.Overlay(ctx)
			if err != nil {
				nodeStream <- nodeStreamMsg{Name: name, Error: err}
				return
			}

			peers, err := node.Peers(ctx)
			if err != nil {
				nodeStream <- nodeStreamMsg{Name: name, Error: err}
				return
			}

			for m := range node.PingStream(ctx, peers) {
				if m.Error != nil {
					nodeStream <- nodeStreamMsg{Name: name, Error: m.Error}
				}
				nodeStream <- nodeStreamMsg{
					Name:        name,
					Address:     address,
					PeerAddress: m.Node,
					RTT:         m.RTT,
				}
			}
		}(k, v)
	}

	go func() {
		wg.Wait()
		close(nodeStream)
	}()

	return nodeStream
}

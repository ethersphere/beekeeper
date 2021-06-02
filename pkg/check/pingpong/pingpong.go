package pingpong

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents check options
type Options struct {
	MetricsPusher *push.Pusher
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		MetricsPusher: nil,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

// Run executes ping check
func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	fmt.Println("running pingpong")
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

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

		for n := range nodeStream(ctx, nodesClients) { // TODO: confirm use case for nodeStream(ctx, ng.NodesClientsAll(ctx))
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

	fmt.Println("pingpong check completed successfully")
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

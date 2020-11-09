package pingpong

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Check executes ping from all nodes to all other nodes in the cluster
func Check(cluster *bee.DynamicCluster, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()

	pusher.Collector(rttGauge)
	pusher.Collector(rttHistogram)

	pusher.Format(expfmt.FmtText)
	nodeGroups := cluster.NodeGroups()
	for _, ng := range nodeGroups {
		for n := range nodeStream(ctx, ng.Nodes()) {
			for t := 0; t < 5; t++ {
				time.Sleep(2 * time.Duration(t) * time.Second)

				if n.Error != nil {
					if t == 4 {
						return fmt.Errorf("node %s: %s", n.Name, n.Error)
					}
					fmt.Printf("node %s: %s\n", n.Name, n.Error)
					continue
				}
				fmt.Printf("Node %s: %s Peer: %s RTT: %s\n", n.Name, n.Address, n.PeerAddress, n.RTT)

				rtt, err := time.ParseDuration(n.RTT)
				if err != nil {
					if t == 4 {
						return fmt.Errorf("node %s: %s", n.Name, err)
					}
					fmt.Printf("node %s: %s\n", n.Name, err)
					continue
				}

				rttGauge.WithLabelValues(n.Address.String(), n.PeerAddress.String()).Set(rtt.Seconds())
				rttHistogram.Observe(rtt.Seconds())

				if pushMetrics {
					if err := pusher.Push(); err != nil {
						fmt.Printf("node %s: %s\n", n.Name, err)
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

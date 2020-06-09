package pingpong

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Check executes ping from all nodes to all other nodes in the cluster
func Check(cluster bee.Cluster, pusher *push.Pusher) (err error) {
	ctx := context.Background()

	pusher.Collector(rttGauge)
	pusher.Collector(rttHistogram)

	for n := range nodeStream(ctx, cluster.Nodes) {
		if n.Error != nil {
			fmt.Printf("node %d: %s\n", n.Index, n.Error)
			continue
		}
		fmt.Printf("Node: %s Peer: %s RTT: %s\n", n.Address, n.PeerAddress, n.RTT)

		rtt, err := time.ParseDuration(n.RTT)
		if err != nil {
			fmt.Printf("node %d: %s\n", n.Index, err)
			continue
		}

		rttGauge.WithLabelValues(n.Address.String(), n.PeerAddress.String()).Set(rtt.Seconds())
		rttHistogram.Observe(rtt.Seconds())

		if err := pusher.Push(); err != nil {
			fmt.Printf("node %d: %s\n", n.Index, err)
		}
	}

	return
}

type nodeStreamMsg struct {
	Index       int
	Address     swarm.Address
	PeerIndex   int
	PeerAddress swarm.Address
	RTT         string
	Error       error
}

func nodeStream(ctx context.Context, nodes []bee.Node) <-chan nodeStreamMsg {
	nodeStream := make(chan nodeStreamMsg)

	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node bee.Node) {
			defer wg.Done()

			address, err := node.Overlay(ctx)
			if err != nil {
				nodeStream <- nodeStreamMsg{Index: i, Error: err}
				return
			}

			peers, err := node.Peers(ctx)
			if err != nil {
				nodeStream <- nodeStreamMsg{Index: i, Error: err}
				return
			}

			for m := range node.PingStream(ctx, peers) {
				if m.Error != nil {
					nodeStream <- nodeStreamMsg{Index: i, Error: m.Error}
				}
				nodeStream <- nodeStreamMsg{
					Index:       i,
					Address:     address,
					PeerIndex:   m.Index,
					PeerAddress: m.Node,
					RTT:         m.RTT,
				}
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(nodeStream)
	}()

	return nodeStream
}

package pingpong

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Check executes ping from all nodes to all other nodes in the cluster
func Check(cluster bee.Cluster) (err error) {
	ctx := context.Background()
	t1 := time.Now()
	var wg sync.WaitGroup
	for i, node := range cluster.Nodes {
		wg.Add(1)
		go func(i int, n bee.Node) {
			defer wg.Done()

			overlay, err := n.Overlay(ctx)
			if err != nil {
				fmt.Printf("node %d: %s\n", i, err)
			}

			peers, err := n.Peers(ctx)
			if err != nil {
				fmt.Printf("node %d: %s\n", i, err)
			}

			var msgs []pingStreamMsg
			for m := range pingStream(ctx, n, peers) {
				msgs = append(msgs, m)
			}

			sort.SliceStable(msgs, func(i, j int) bool {
				return msgs[i].Index < msgs[j].Index
			})

			for j, m := range msgs {
				if m.Error != nil {
					fmt.Printf("node %d had error pinging peer %s: %s\n", i, peers[j], m.Error)
				}
				fmt.Printf("Node %d. Peer %d RTT: %s. Node: %s Peer: %s \n", i, j, m.RTT, overlay, peers[j])
			}
		}(i, node)
	}
	wg.Wait()

	fmt.Println("Elapsed: ", time.Since(t1))

	return
}

type pingStreamMsg struct {
	RTT   string
	Index int
	Error error
}

func pingStream(ctx context.Context, node bee.Node, peers []swarm.Address) <-chan pingStreamMsg {
	pingStream := make(chan pingStreamMsg)

	var wg sync.WaitGroup
	for i, p := range peers {
		wg.Add(1)
		go func(i int, n bee.Node, p swarm.Address) {
			defer wg.Done()
			rtt, err := n.Ping(ctx, p)
			pingStream <- pingStreamMsg{
				RTT:   rtt,
				Index: i,
				Error: err,
			}
		}(i, node, p)
	}

	go func() {
		wg.Wait()
		close(pingStream)
	}()

	return pingStream
}

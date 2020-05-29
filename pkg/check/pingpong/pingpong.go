package pingpong

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// SHOULD BEEKEEPER BE AWARE OF THE OREDER
// SHOULD BEEKEEPER BE AWARE OF THE OREDER
// SHOULD BEEKEEPER BE AWARE OF THE OREDER

// Check executes ping from all nodes to all other nodes in the cluster
func Check(cluster bee.Cluster) (err error) {
	ctx := context.Background()
	var result []nodeResult

	var wg sync.WaitGroup
	for i, node := range cluster.Nodes {
		wg.Add(1)
		go func(i int, n bee.Node) {
			defer wg.Done()

			address, err := n.Overlay(ctx)
			if err != nil {
				result = append(result, nodeResult{
					Index: i,
					Error: err,
				})
				return
			}

			peers, err := n.Peers(ctx)
			if err != nil {
				result = append(result, nodeResult{
					Index: i,
					Error: err,
				})
				return
			}

			var pingResults []pingStreamMsg
			for m := range pingStream(ctx, n, peers) {
				pingResults = append(pingResults, m)
			}
			sort.SliceStable(pingResults, func(i, j int) bool {
				return pingResults[i].Index < pingResults[j].Index
			})

			result = append(result, nodeResult{
				Index:       i,
				Address:     address,
				PingResults: pingResults,
			})

			return
		}(i, node)
	}
	wg.Wait()

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Index < result[j].Index
	})

	for i, n := range result {
		if n.Error != nil {
			fmt.Printf("node %d: %s\n", i, n.Error)
			continue
		}
		for j, p := range n.PingResults {
			if p.Error != nil {
				fmt.Printf("node %d had error pinging peer %d: %s\n", i, j, p.Error)
			}
			fmt.Printf("Node %d. Peer %d RTT: %s. Node: %s Peer: %s\n", i, j, p.RTT, n.Address, p.Address)
		}
	}

	return
}

type nodeResult struct {
	Index       int
	Address     swarm.Address
	PingResults []pingStreamMsg
	Error       error
}

type pingStreamMsg struct {
	Index   int
	Address swarm.Address
	RTT     string
	Error   error
}

func pingStream(ctx context.Context, node bee.Node, peers []swarm.Address) <-chan pingStreamMsg {
	pingStream := make(chan pingStreamMsg)

	var wg sync.WaitGroup
	for i, peer := range peers {
		wg.Add(1)
		go func(n bee.Node, i int, p swarm.Address) {
			defer wg.Done()
			rtt, err := n.Ping(ctx, p)
			pingStream <- pingStreamMsg{
				Index:   i,
				Address: p,
				RTT:     rtt,
				Error:   err,
			}
		}(node, i, peer)
	}

	go func() {
		wg.Wait()
		close(pingStream)
	}()

	return pingStream
}

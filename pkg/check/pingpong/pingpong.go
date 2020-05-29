package pingpong

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Check executes ping from all nodes to all other nodes in the cluster
func Check(cluster bee.Cluster) (err error) {
	ctx := context.Background()
	for i, n := range cluster.Nodes {
		o, err := n.Overlay(ctx)
		if err != nil {
			return fmt.Errorf("error getting overlay for node %d: %w", i, err)
		}

		peers, err := n.Peers(ctx)
		if err != nil {
			return fmt.Errorf("error getting peers for node %d: %w", i, err)
		}

		for j, p := range peers {
			rtt, err := n.Ping(ctx, p)
			if err != nil {
				return fmt.Errorf("node %d had error pinging peer %s: %w", i, p.String(), err)
			}
			fmt.Printf("Node %d. Peer %d RTT: %s. Node: %s Peer: %s \n", i, j, rtt, o.String(), p.String())
		}
	}

	return
}

package peercount

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Check executes peer count check on cluster
func Check(cluster bee.Cluster) (err error) {
	var expectedPeerCount = cluster.Size() - 1

	ctx := context.Background()
	for i, n := range cluster.Nodes {
		o, err := n.Overlay(ctx)
		if err != nil {
			return fmt.Errorf("node %d: %w", i, err)
		}

		peers, err := n.Peers(ctx)
		if err != nil {
			return fmt.Errorf("node %d: %w", i, err)
		}

		fmt.Printf("Node %d. Peers %d/%d. Node: %s\n", i, len(peers), expectedPeerCount, o.String())
	}

	return
}

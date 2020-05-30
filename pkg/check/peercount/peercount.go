package peercount

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Check executes peer count check on cluster
func Check(cluster bee.Cluster) (err error) {
	ctx := context.Background()

	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return err
	}

	peers, err := cluster.Peers(ctx)
	if err != nil {
		return err
	}

	clusterSize := cluster.Size()
	for i := 0; i < clusterSize; i++ {
		fmt.Printf("Node %d. Peers %d/%d. Node: %s\n", i, len(peers[i]), clusterSize-1, overlays[i])
	}

	return
}

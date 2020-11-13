package peercount

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Check executes peer count check on cluster
func Check(cluster *bee.Cluster) (err error) {
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
	for g, v := range peers {
		for n, p := range v {
			fmt.Printf("Node %s. Peers %d/%d. Address: %s\n", n, len(p), clusterSize-1, overlays[g][n])
		}
	}

	return
}

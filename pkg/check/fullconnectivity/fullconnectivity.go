package fullconnectivity

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

var errFullConnectivity = errors.New("full connectivity")

// Check executes full connectivity check if cluster is fully connected
func Check(ctx context.Context, cluster *bee.Cluster) (err error) {
	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return err
	}

	peers, err := cluster.Peers(ctx)
	if err != nil {
		return err
	}

	clusterSize := cluster.Size()
	expectedPeerCount := clusterSize - 1

	for group, v := range overlays {
		for node, overlay := range v {
			if len(peers[group][node]) != expectedPeerCount {
				fmt.Printf("Node %s. Failed. Peers %d/%d. Address: %s\n", node, len(peers[group][node]), expectedPeerCount, overlay)
				return errFullConnectivity
			}

			for _, p := range peers[group][node] {
				if !contains(overlays, p) {
					fmt.Printf("Node %s. Failed. Invalid peer: %s. Node: %s\n", node, p.String(), overlay)
					return errFullConnectivity
				}
			}

			fmt.Printf("Node %s. Passed. Peers %d/%d. All peers are valid. Node: %s\n", node, len(peers[group][node]), expectedPeerCount, overlay)
		}
	}

	return
}

// contains checks if a given set of swarm.Address contains given swarm.Address
func contains(s bee.ClusterOverlays, a swarm.Address) bool {
	for _, v := range s {
		for _, o := range v {
			if o.Equal(a) {
				return true
			}
		}
	}

	return false
}

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
	expectedPeerCount := clusterSize - 1

	for i := 0; i < clusterSize; i++ {
		if len(peers[i]) != expectedPeerCount {
			fmt.Printf("Node %d. Failed. Peers %d/%d. Node: %s\n", i, len(peers[i]), expectedPeerCount, overlays[i])
			return errFullConnectivity
		}

		for _, p := range peers[i] {
			if !contains(overlays, p) {
				fmt.Printf("Node %d. Failed. Invalid peer: %s. Node: %s\n", i, p.String(), overlays[i])
				return errFullConnectivity
			}
		}

		fmt.Printf("Node %d. Passed. Peers %d/%d. All peers are valid. Node: %s\n", i, len(peers[i]), expectedPeerCount, overlays[i])
	}

	return
}

// contains checks if a given set of swarm.Address contains given swarm.Address
func contains(s []swarm.Address, v swarm.Address) bool {
	for _, a := range s {
		if a.Equal(v) {
			return true
		}
	}
	return false
}

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

	var expectedPeerCount = cluster.Size() - 1
	for i, n := range cluster.Nodes {
		peers, err := n.Peers(ctx)
		if err != nil {
			return err
		}

		if len(peers) != expectedPeerCount {
			fmt.Printf("Node %d failed. Peers %d/%d.\n", i, len(peers), expectedPeerCount)
			return errFullConnectivity
		}

		for _, p := range peers {
			if !contains(overlays, p) {
				fmt.Printf("Node %d failed. Invalid peer: %s\n", i, p.String())
				return errFullConnectivity
			}
		}

		fmt.Printf("Node %d passed. Peers %d/%d. All peers are valid. Overlay %s.\n", i, len(peers), expectedPeerCount, overlays[i])
	}

	return
}

// contains checks if a given set of swarm.Address containes given swarm.Address
func contains(s []swarm.Address, v swarm.Address) bool {
	for _, a := range s {
		if a.Equal(v) {
			return true
		}
	}
	return false
}

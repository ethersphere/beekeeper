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
			return fmt.Errorf("node %d: %w", i, err)
		}

		if len(peers) != expectedPeerCount {
			fmt.Printf("Node %d. Failed. Peers %d/%d. Node: %s\n", i, len(peers), expectedPeerCount, overlays[i].String())
			return errFullConnectivity
		}

		for _, p := range peers {
			if !contains(overlays, p) {
				fmt.Printf("Node %d. Failed. Invalid peer: %s. Node: %s\n", i, p.String(), overlays[i].String())
				return errFullConnectivity
			}
		}

		fmt.Printf("Node %d. Passed. Peers %d/%d. All peers are valid. Node: %s\n", i, len(peers), expectedPeerCount, overlays[i].String())
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

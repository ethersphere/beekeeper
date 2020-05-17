package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

var errFullConnectivity = errors.New("full connectivity")

// FullConnectivity ...
func FullConnectivity(cluster bee.Cluster) (err error) {
	ctx := context.Background()

	var overlays []swarm.Address
	for _, n := range cluster.Nodes {
		a, err := n.Debug().Node.Addresses(ctx)
		if err != nil {
			return err
		}

		overlays = append(overlays, a.Overlay)
	}

	var expectedPeerCount = cluster.Size() - 1
	for i, n := range cluster.Nodes {
		p, err := n.Debug().Node.Peers(ctx)
		if err != nil {
			return err
		}

		if len(p.Peers) != expectedPeerCount {
			fmt.Printf("Node %d failed. Peers %d/%d.\n", i, len(p.Peers), expectedPeerCount)
			return errFullConnectivity
		}

		for _, p := range p.Peers {
			if !contains(overlays, p.Address) {
				fmt.Printf("Node %d failed. Invalid peer: %s\n", i, p.Address)
				return errFullConnectivity
			}
		}

		fmt.Printf("Node %d passed. Peers %d/%d. All peers are valid. Overlay %s.\n", i, len(p.Peers), expectedPeerCount, overlays[i])
	}

	return
}

// contains checks if slice of swarm.Address containes given swarm.Address
func contains(s []swarm.Address, v swarm.Address) bool {
	for _, a := range s {
		if a.Equal(v) {
			return true
		}
	}
	return false
}

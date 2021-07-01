package fullconnectivity

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
)

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

var errFullConnectivity = errors.New("full connectivity")

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	if err := checkFullNodesConnectivity(ctx, cluster); err != nil {
		return fmt.Errorf("check full nodes: %w", err)
	}
	if err := checkLightNodesConnectivity(ctx, cluster); err != nil {
		return fmt.Errorf("check light nodes: %w", err)
	}

	return
}

func checkFullNodesConnectivity(ctx context.Context, cluster *bee.Cluster) (err error) {
	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return err
	}

	fullNodes, err := cluster.Overlays(ctx, "light")
	if err != nil {
		return err
	}

	peers, err := cluster.Peers(ctx, "light")
	if err != nil {
		return err
	}

	clusterSize := cluster.Size()
	expectedPeerCount := clusterSize - len(cluster.LightNodeNames()) - 1 // we expect to be connected to all full nodes

	for group, v := range fullNodes {
		for node, overlay := range v {
			allPeers := peers[group][node]

			if len(allPeers) < expectedPeerCount {
				fmt.Printf("Node %s. Failed. Peers %d/%d. Address: %s\n", node, len(allPeers), expectedPeerCount, overlay)
				return errFullConnectivity
			}

			for _, p := range allPeers {
				if !contains(overlays, p) {
					fmt.Printf("Node %s. Failed. Invalid peer: %s. Node: %s\n", node, p.String(), overlay)
					return errFullConnectivity
				}
			}

			fmt.Printf("Node %s. Passed. Peers %d/%d. All peers are valid. Node: %s\n", node, len(allPeers), expectedPeerCount, overlay)
		}
	}

	return
}

func checkLightNodesConnectivity(ctx context.Context, cluster *bee.Cluster) (err error) {
	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return err
	}

	lightNodes, err := cluster.Overlays(ctx, "bee", "bootnode")
	if err != nil {
		return err
	}

	peers, err := cluster.Peers(ctx, "bee", "bootnode")
	if err != nil {
		return err
	}

	for group, v := range lightNodes {
		for node, overlay := range v {
			allPeers := peers[group][node]

			if len(allPeers) < 1 { // expected to be connected to the bootnode
				fmt.Printf("Node %s. Failed. Peers %d/%d. Address: %s\n", node, len(allPeers), 1, overlay)
				return errFullConnectivity
			}

			for _, p := range allPeers {
				if !contains(overlays, p) {
					fmt.Printf("Node %s. Failed. Invalid peer: %s. Node: %s\n", node, p.String(), overlay)
					return errFullConnectivity
				}
			}

			fmt.Printf("Node %s. Passed. Peers %d/%d. All peers are valid. Node: %s\n", node, len(allPeers), 1, overlay)
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

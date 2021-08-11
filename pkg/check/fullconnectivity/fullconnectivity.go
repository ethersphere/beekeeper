package fullconnectivity

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	orchestration "github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
)

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

type Options struct {
	LightNodeNames []string
	FullNodeNames  []string
	BootNodeNames  []string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{}
}

var errFullConnectivity = errors.New("full connectivity")

func (c *Check) Run(ctx context.Context, cluster *orchestration.Cluster, opts interface{}) (err error) {
	lightNodes := opts.(Options).LightNodeNames
	bootNodes := opts.(Options).BootNodeNames
	if err := checkFullNodesConnectivity(ctx, cluster, lightNodes, bootNodes); err != nil {
		return fmt.Errorf("check full nodes: %w", err)
	}
	fullNodes := opts.(Options).FullNodeNames
	if err := checkLightNodesConnectivity(ctx, cluster, fullNodes); err != nil {
		return fmt.Errorf("check light nodes: %w", err)
	}

	return
}

func checkFullNodesConnectivity(ctx context.Context, cluster *orchestration.Cluster, skipNodes, bootNodes []string) (err error) {
	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return err
	}

	fullNodes, err := cluster.Overlays(ctx, skipNodes...)
	if err != nil {
		return err
	}

	peers, err := cluster.Peers(ctx, skipNodes...)
	if err != nil {
		return err
	}

	fullNodeNames := cluster.FullNodeNames()
	fullNodeCount := len(fullNodeNames) - 1 // we expect to be connected to all full nodes except self

	for group, v := range fullNodes {
		expectedPeerCount := fullNodeCount
		if isBootNode(group, bootNodes) {
			expectedPeerCount = len(cluster.NodeNames()) - 1 // bootnodes are connected to all others
		}
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

func isBootNode(group string, bootnodes []string) bool {
	for _, b := range bootnodes {
		if group == b {
			return true
		}
	}

	return false
}

func checkLightNodesConnectivity(ctx context.Context, cluster *orchestration.Cluster, skipNodes []string) (err error) {
	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return err
	}

	lightNodes, err := cluster.Overlays(ctx, skipNodes...)
	if err != nil {
		return err
	}

	peers, err := cluster.Peers(ctx, skipNodes...)
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
func contains(s orchestration.ClusterOverlays, a swarm.Address) bool {
	for _, v := range s {
		for _, o := range v {
			if o.Equal(a) {
				return true
			}
		}
	}

	return false
}

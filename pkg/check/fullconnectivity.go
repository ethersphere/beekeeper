package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// FullConnectivityOptions ...
type FullConnectivityOptions struct {
	APIHostnamePattern      string
	APIDomain               string
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	DisableNamespace        bool
	Namespace               string
	NodeCount               int
}

var errFullConnectivity = errors.New("full connectivity")

// FullConnectivity ...
func FullConnectivity(opts FullConnectivityOptions) (err error) {
	var expectedPeerCount = opts.NodeCount - 1
	ctx := context.Background()

	nodes, err := bee.NewNNodes(opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, opts.DisableNamespace, opts.NodeCount)
	if err != nil {
		return err
	}

	var overlays []swarm.Address
	for _, n := range nodes {
		a, err := n.DebugAPI.Node.Addresses(ctx)
		if err != nil {
			return err
		}

		overlays = append(overlays, a.Overlay)
	}

	for i, n := range nodes {
		p, err := n.DebugAPI.Node.Peers(ctx)
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

		fmt.Printf("Node %d passed. Peers %d/%d. All peers are valid.\n", i, len(p.Peers), expectedPeerCount)
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

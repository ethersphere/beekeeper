package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
)

// FullConnectivityOptions ...
type FullConnectivityOptions struct {
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

	var nodes []node
	var overlays []swarm.Address
	for i := 0; i < opts.NodeCount; i++ {
		debugAPIURL, err := createURL(scheme, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, i, opts.DisableNamespace)
		if err != nil {
			return err
		}

		dc := debugapi.NewClient(debugAPIURL, nil)
		ctx := context.Background()

		a, err := dc.Node.Addresses(ctx)
		if err != nil {
			return err
		}

		p, err := dc.Node.Peers(ctx)
		if err != nil {
			return err
		}

		nodes = append(nodes, node{
			Addresses: a,
			Peers:     p,
		})
		overlays = append(overlays, a.Overlay)
	}

	for i, n := range nodes {
		if len(n.Peers.Peers) != expectedPeerCount {
			fmt.Printf("Node %d failed. Peers %d/%d.\n", i, len(n.Peers.Peers), expectedPeerCount)
			return errFullConnectivity
		}

		for _, p := range n.Peers.Peers {
			if !contains(overlays, p.Address) {
				fmt.Printf("Node %d failed. Invalid peer: %s\n", i, p.Address)
				return errFullConnectivity
			}
		}

		fmt.Printf("Node %d passed. Peers %d/%d. All peers are valid.\n", i, len(n.Peers.Peers), expectedPeerCount)
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

package check

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// PingPongOptions ...
type PingPongOptions struct {
	APIHostnamePattern      string
	APIDomain               string
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	DisableNamespace        bool
	Namespace               string
	NodeCount               int
}

// PingPong ...
func PingPong(opts PingPongOptions) (err error) {
	ctx := context.Background()

	nodes, err := bee.NewNNodes(opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, opts.DisableNamespace, opts.NodeCount)
	if err != nil {
		return err
	}

	for i, n := range nodes {
		a, err := n.DebugAPI.Node.Addresses(ctx)
		if err != nil {
			return err
		}

		p, err := n.DebugAPI.Node.Peers(ctx)
		if err != nil {
			return err
		}

		for j, peer := range p.Peers {
			r, err := n.API.PingPong.Ping(ctx, peer.Address)
			if err != nil {
				return err
			}
			fmt.Printf("RTT %s. Node %d - Peer %d. %s - %s. \n", r.RTT, i, j, a.Overlay.String(), peer.Address)
		}
	}

	return
}

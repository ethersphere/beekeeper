package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// PeerCountOptions ...
type PeerCountOptions struct {
	APIHostnamePattern      string
	APIDomain               string
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	DisableNamespace        bool
	Namespace               string
	NodeCount               int
}

var errPeerCount = errors.New("peer count")

// PeerCount ...
func PeerCount(opts PeerCountOptions) (err error) {
	var expectedPeerCount = opts.NodeCount - 1
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

		if len(p.Peers) == expectedPeerCount {
			fmt.Printf("Node %d passed. Peers %d/%d. Overlay %s.\n", i, len(p.Peers), expectedPeerCount, a.Overlay.String())
		} else {
			fmt.Printf("Node %d failed. Peers %d/%d. Overlay %s.\n", i, len(p.Peers), expectedPeerCount, a.Overlay.String())
			return errPeerCount
		}
	}

	return
}

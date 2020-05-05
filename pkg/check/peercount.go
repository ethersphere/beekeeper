package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
)

// PeerCountOptions ...
type PeerCountOptions struct {
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	Namespace               string
	NodeCount               int
}

var errPeerCount = errors.New("peer count")

// PeerCount ...
func PeerCount(opts PeerCountOptions) (err error) {
	var expectedPeerCount = opts.NodeCount - 1

	for i := 0; i < opts.NodeCount; i++ {
		debugAPIURL, err := createURL(scheme, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, i)
		if err != nil {
			return err
		}

		dc := debugapi.NewClient(debugAPIURL, nil)
		ctx := context.Background()

		resp, err := dc.Node.Peers(ctx)
		if err != nil {
			return err
		}

		if len(resp.Peers) == expectedPeerCount {
			fmt.Printf("Node %d passed. Peers %d/%d.\n", i, len(resp.Peers), expectedPeerCount)
		} else {
			fmt.Printf("Node %d failed. Peers %d/%d.\n", i, len(resp.Peers), expectedPeerCount)
			return errPeerCount
		}
	}

	return
}

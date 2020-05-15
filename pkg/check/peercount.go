package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// PeerCount ...
func PeerCount(nodes []bee.Node) (err error) {
	var expectedPeerCount = len(nodes) - 1

	ctx := context.Background()
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
			return errors.New("peer count")
		}
	}

	return
}

package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// PeerCount checks cluster's peer count
func PeerCount(cluster bee.Cluster) (err error) {
	var expectedPeerCount = cluster.Size() - 1

	ctx := context.Background()
	for i, n := range cluster.Nodes {
		o, err := n.Overlay(ctx)
		if err != nil {
			return err
		}

		peers, err := n.Peers(ctx)
		if err != nil {
			return err
		}

		if len(peers) == expectedPeerCount {
			fmt.Printf("Node %d passed. Peers %d/%d. Overlay %s.\n", i, len(peers), expectedPeerCount, o.String())
		} else {
			fmt.Printf("Node %d failed. Peers %d/%d. Overlay %s.\n", i, len(peers), expectedPeerCount, o.String())
			return errors.New("peer count")
		}
	}

	return
}

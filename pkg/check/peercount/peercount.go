package peercount

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

var errPeerCount = errors.New("peer count")

// Check executes peer count check on cluster
func Check(cluster bee.Cluster) (err error) {
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
			fmt.Printf("Node %d. Passed. Peers %d/%d. Node: %s\n", i, len(peers), expectedPeerCount, o.String())
		} else {
			fmt.Printf("Node %d. Failed. Peers %d/%d. Node: %s\n", i, len(peers), expectedPeerCount, o.String())
			return errPeerCount
		}
	}

	return
}

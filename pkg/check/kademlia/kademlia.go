package kademlia

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Check ...
func Check(cluster bee.Cluster) (err error) {
	ctx := context.Background()

	for i, n := range cluster.Nodes {
		t, err := n.Topology(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("CHECK %d %+v\n", i, t)
	}

	return
}

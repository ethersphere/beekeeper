package pingpong

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Check executes ping from all nodes to all other nodes in the cluster
func Check(cluster bee.Cluster) (err error) {
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

		for j, p := range peers {
			rtt, err := n.Ping(ctx, p)
			if err != nil {
				return err
			}
			fmt.Printf("RTT %s. Node %d - Peer %d. %s - %s. \n", rtt, i, j, o.String(), p.String())
		}
	}

	return
}

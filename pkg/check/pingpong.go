package check

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// PingPong ...
func PingPong(cluster bee.Cluster) (err error) {
	ctx := context.Background()
	for i, n := range cluster.Nodes {
		a, err := n.Debug().Node.Addresses(ctx)
		if err != nil {
			return err
		}

		p, err := n.Debug().Node.Peers(ctx)
		if err != nil {
			return err
		}

		for j, peer := range p.Peers {
			rtt, err := n.Ping(ctx, peer.Address)
			if err != nil {
				return err
			}
			fmt.Printf("RTT %s. Node %d - Peer %d. %s - %s. \n", rtt, i, j, a.Overlay.String(), peer.Address)
		}
	}

	return
}

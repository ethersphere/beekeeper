package check

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// PingPong ...
func PingPong(nodes []bee.Node) (err error) {
	ctx := context.Background()
	for i, n := range nodes {
		a, err := n.DebugAPI().Node.Addresses(ctx)
		if err != nil {
			return err
		}

		p, err := n.DebugAPI().Node.Peers(ctx)
		if err != nil {
			return err
		}

		for j, peer := range p.Peers {
			r, err := n.API().PingPong.Ping(ctx, peer.Address)
			if err != nil {
				return err
			}
			fmt.Printf("RTT %s. Node %d - Peer %d. %s - %s. \n", r.RTT, i, j, a.Overlay.String(), peer.Address)
		}
	}

	return
}

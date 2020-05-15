package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
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

var errPingPong = errors.New("ping pong")

// PingPong ...
func PingPong(opts PingPongOptions) (err error) {
	ctx := context.Background()

	for i := 0; i < opts.NodeCount; i++ {
		n, err := bee.NewNode(opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, i, opts.DisableNamespace)
		if err != nil {
			fmt.Println(1)
			return err
		}

		p, err := n.DebugAPI.Node.Peers(ctx)
		if err != nil {
			fmt.Println(2)
			return err
		}

		APIURL, err := createURL(scheme, opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, i, opts.DisableNamespace)
		if err != nil {
			return err
		}
		c := api.NewClient(APIURL, nil)

		for j, peer := range p.Peers {
			r, err := c.PingPong.Ping(ctx, peer.Address)
			// r, err := n.API.PingPogng.Ping(ctx, peer.Address)
			if err != nil {
				return err
			}
			fmt.Printf("RTT [Node %d - Peer %d]: %s\n", i, j, r.RTT)
		}
	}

	return
}

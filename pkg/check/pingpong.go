package check

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
)

// PingPongOptions ...
type PingPongOptions struct {
	APIHostnamePattern      string
	APIDomain               string
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	Namespace               string
	NodeCount               int
}

// PingPong ...
func PingPong(opts PingPongOptions) (err error) {
	for i := 0; i < opts.NodeCount; i++ {
		debugAPI, err := nodeURL(scheme, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, i)
		if err != nil {
			return err
		}

		bc := debugapi.NewClient(debugAPI, nil)
		ctx := context.Background()

		resp, err := bc.Node.Peers(ctx)
		if err != nil {
			return err
		}

		API, err := nodeURL(scheme, opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, i)
		if err != nil {
			return err
		}

		c := api.NewClient(API, nil)
		ctx = context.Background()

		for j, p := range resp.Peers {
			resp, err := c.PingPong.Ping(ctx, p.Address)
			if err != nil {
				return err
			}
			fmt.Printf("RTT [Node %d - Peer %d]: %s\n", i, j, resp.RTT)
		}
	}

	return
}

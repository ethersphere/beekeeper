package check

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
)

// PingPongOptions ...
type PingPongOptions struct {
	DebugAPIURLTemplate string
	Namespace           string
	NodeCount           int
}

// PingPong ...
func PingPong(opts PingPongOptions) (err error) {
	for i := 0; i < opts.NodeCount; i++ {
		var debugAPIURL *url.URL
		var err error
		if opts.DebugAPIURLTemplate != "" {
			debugAPIURL, err = url.Parse(fmt.Sprintf(opts.DebugAPIURLTemplate, i))
		} else {
			debugAPIURL, err = url.Parse(fmt.Sprintf(debugAPIURLTemplate, i, opts.Namespace))
		}
		if err != nil {
			return err
		}

		bc := debugapi.NewClient(debugAPIURL, nil)
		ctx := context.Background()

		resp, err := bc.Node.Peers(ctx)
		if err != nil {
			return err
		}

		APIURL, err := url.Parse(fmt.Sprintf(apiURLTemplate, i, opts.Namespace))
		if err != nil {
			return err
		}
		c := api.NewClient(APIURL, nil)
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

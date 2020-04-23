package check

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
)

// PeerCountOptions ...
type PeerCountOptions struct {
	NodeCount       int
	NodeURLTemplate string
}

var errPeerCount = errors.New("peer count")

// PeerCount ...
func PeerCount(opts PeerCountOptions) (err error) {
	var resp debugapi.Peers

	for i := 0; i < opts.NodeCount; i++ {
		nodeURL, err := url.Parse(fmt.Sprintf(opts.NodeURLTemplate, i))
		if err != nil {
			return err
		}

		bc := debugapi.NewClient(nodeURL, nil)
		ctx := context.Background()

		resp, err = bc.Node.Peers(ctx)
		if err != nil {
			return err
		}

		if len(resp.Peers) == opts.NodeCount {
			fmt.Println(fmt.Sprintf("Node %d passed. Peers %d/%d.", i, len(resp.Peers), opts.NodeCount))
		} else {
			fmt.Println(fmt.Sprintf("Node %d failed. Peers %d/%d.", i, len(resp.Peers), opts.NodeCount))
			return errPeerCount
		}
	}

	return
}

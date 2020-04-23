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
	BootNodeCount   int
	NodeCount       int
	NodeURLTemplate string
}

var errPeerCount = errors.New("peer count")

// PeerCount ...
func PeerCount(opts PeerCountOptions) (err error) {
	var expectedPeerCount = opts.NodeCount + opts.BootNodeCount - 1

	for i := 0; i < opts.NodeCount; i++ {
		nodeURL, err := url.Parse(fmt.Sprintf(opts.NodeURLTemplate, i))
		if err != nil {
			return err
		}

		bc := debugapi.NewClient(nodeURL, nil)
		ctx := context.Background()

		resp, err := bc.Node.Peers(ctx)
		if err != nil {
			return err
		}

		if len(resp.Peers) == expectedPeerCount {
			fmt.Println(fmt.Sprintf("Node %d passed. Peers %d/%d.", i, len(resp.Peers), expectedPeerCount))
		} else {
			fmt.Println(fmt.Sprintf("Node %d failed. Peers %d/%d.", i, len(resp.Peers), expectedPeerCount))
			return errPeerCount
		}
	}

	return
}

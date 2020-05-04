package check

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
)

const debugAPIURLTemplate = "http://bee-%d-debug.%s.core.internal"

// PeerCountOptions ...
type PeerCountOptions struct {
	NodeCount   int
	Namespace   string
	URLTemplate string
}

var errPeerCount = errors.New("peer count")

// PeerCount ...
func PeerCount(opts PeerCountOptions) (err error) {
	var expectedPeerCount = opts.NodeCount - 1

	for i := 0; i < opts.NodeCount; i++ {
		var nodeURL *url.URL
		var err error
		if opts.URLTemplate != "" {
			nodeURL, err = url.Parse(fmt.Sprintf(opts.URLTemplate, i))
		} else {
			nodeURL, err = url.Parse(fmt.Sprintf(debugAPIURLTemplate, i, opts.Namespace))
		}
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
			fmt.Printf("Node %d passed. Peers %d/%d.\n", i, len(resp.Peers), expectedPeerCount)
		} else {
			fmt.Printf("Node %d failed. Peers %d/%d.\n", i, len(resp.Peers), expectedPeerCount)
			return errPeerCount
		}
	}

	return
}

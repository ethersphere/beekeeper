package peercount

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return err
	}

	peers, err := cluster.Peers(ctx)
	if err != nil {
		return err
	}

	clusterSize := cluster.Size()
	for g, v := range peers {
		for n, p := range v {
			fmt.Printf("Node %s. Peers %d/%d. Address: %s\n", n, len(p), clusterSize-1, overlays[g][n])
		}
	}

	return
}

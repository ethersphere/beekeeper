package peercount

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
)

// compile check whether Check implements interface
var _ check.Check = (*Check2)(nil)

// TODO: rename to Check
// Check instance
type Check2 struct{}

// NewCheck returns new check
func NewCheck() check.Check {
	return &Check2{}
}

// Options represents check options
type Options struct {
	MetricsEnabled bool
	Seed           int64
}

func (c *Check2) Run(ctx context.Context, cluster *bee.Cluster, o interface{}) (err error) {
	return
}

// Check executes peer count check on cluster
func Check(cluster *bee.Cluster) (err error) {
	ctx := context.Background()

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

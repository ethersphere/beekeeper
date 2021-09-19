package chunkavailability

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents check options
type Options struct {
	Seed int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		Seed: random.Int64(),
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{
		metrics: newMetrics(),
	}
}

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, metricsPusher *push.Pusher, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if metricsPusher != nil {
		mm.RegisterCollectors(c.Metrics()...)
	}

	defer func() {
		if err != nil {
			c.metrics.Failures.Inc()
		}
	}()

	//var (
	//rnds = random.PseudoGenerator(o.Seed)
	//)

	fmt.Printf("seed: %d\n", o.Seed)

	//overlays, err := cluster.FlattenOverlays(ctx)
	//if err != nil {
	//return err
	//}

	//clients, err := cluster.NodesClients(ctx)
	//if err != nil {
	//return err
	//}

	/*
		- pick a random node
		- upload chunk
		- check option of number of nodes to download from
		- download from nodes
		- mark histograms of how long it takes for chunk to be available from all nodes

	*/
	//batchID, err := uploader.GetOrCreateBatch(ctx, 10000, 17, "", "")
	//if err != nil {
	//return fmt.Errorf("created batch id %w", err)
	//}

	return nil
}

// findName returns node name of a given swarm.Address in a given set of swarm.Addresses, or "" if not found
func findName(nodes map[string]swarm.Address, addr swarm.Address) (string, bool) {
	for n, a := range nodes {
		if addr.Equal(a) {
			return n, true
		}
	}

	return "", false
}

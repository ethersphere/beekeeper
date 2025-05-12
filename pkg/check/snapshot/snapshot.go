package snapshot

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
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

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(o.Seed)
	clients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("node clients shuffle: %w", err)
	}

	if len(clients) < 2 {
		return fmt.Errorf("not enough nodes to run manifest check")
	}

	client1 := clients[0]
	client2 := clients[1]

	batches1, err := client1.Batches(ctx)
	if err != nil {
		return fmt.Errorf("get batches: %w", err)
	}

	batches2, err := client2.Batches(ctx)
	if err != nil {
		return fmt.Errorf("get batches: %w", err)
	}

	equal, err := c.compareBatches(batches1, batches2)
	if err != nil {
		return fmt.Errorf("compare batches: %w", err)
	}

	if !equal {
		return fmt.Errorf("batches are not equal")
	}

	c.logger.Infof("batches are equal")

	return nil
}

func (c *Check) compareBatches(batches1, batches2 map[string]api.Batch) (bool, error) {
	if len(batches1) != len(batches2) {
		return false, fmt.Errorf("batches length mismatch: %d != %d", len(batches1), len(batches2))
	}

	if len(batches1) == 0 {
		c.logger.Warning("both nodes have no batches")
		return true, nil
	}

	if reflect.DeepEqual(batches1, batches2) {
		return true, nil
	}

	for i := range batches1 {
		if batches1[i].BatchID != batches2[i].BatchID {
			return false, fmt.Errorf("batch ID mismatch: %s != %s", batches1[i].BatchID, batches2[i].BatchID)
		}
	}

	return true, nil
}

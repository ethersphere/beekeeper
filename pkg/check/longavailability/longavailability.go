package longavailability

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

type Options struct {
	Refs       []string
	RndSeed    int64
	RetryCount int
	RetryWait  time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		RndSeed:    time.Now().UnixNano(),
		RetryCount: 3,
		RetryWait:  10 * time.Second,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger:  logger,
		metrics: newMetrics("check_longavailability"),
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o interface{}) error {
	opts, ok := o.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	var addresses []swarm.Address
	for _, ref := range opts.Refs {
		addr, err := swarm.ParseHexAddress(ref)
		if err != nil {
			return fmt.Errorf("parse hex address: %w", err)
		}
		addresses = append(addresses, addr)
	}

	for _, addr := range addresses {
		node, err := findRandomNode(ctx, addr, cluster, opts.RndSeed)
		if err != nil {
			c.logger.Errorf("find node %s. Skipping. %w", addr.String(), err)
			continue
		}

		var i int
		for i = 0; i < opts.RetryCount; i++ {
			c.metrics.DownloadAttempts.Inc()
			c.logger.Infof("node %s: download attempt %d for %s", node.Name(), i+1, addr)

			start := time.Now()
			_, _, err = node.Client().DownloadFile(ctx, addr)
			if err != nil {
				c.metrics.DownloadErrors.Inc()
				c.logger.Errorf("node %s: download %s error: %v", node.Name(), addr, err)
				c.logger.Infof("retrying in: %v", opts.RetryWait)
				time.Sleep(opts.RetryWait)
				continue
			}
			dur := time.Since(start)
			c.metrics.DownloadDuration.Observe(dur.Seconds())
			c.logger.Infof("node %s: downloaded %s successfully in %v", node.Name(), addr, dur)
			break
		}

		if i >= opts.RetryCount {
			c.logger.Errorf("node %s: download for %s failed: %v", node.Name(), addr, err)
		}
	}
	return nil
}

func findRandomNode(ctx context.Context, addr swarm.Address, cluster orchestration.Cluster, randSeed int64) (orchestration.Node, error) {
	nodes := cluster.Nodes()
	var eligible []string
	for name, node := range nodes {
		pins, err := node.Client().GetPins(ctx)
		if err != nil {
			return nil, fmt.Errorf("node %s: get pins: %w", name, err)
		}
		var found bool
		for _, pin := range pins {
			if pin.Equal(addr) {
				found = true
				break
			}
		}
		if !found {
			eligible = append(eligible, name)
		}
	}

	rnd := random.PseudoGenerator(randSeed)
	node := nodes[eligible[rnd.Intn(len(eligible))]]
	return node, nil
}

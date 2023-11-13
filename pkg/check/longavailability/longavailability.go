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
	Refs         []string
	RndSeed      int64
	RetryCount   int
	RetryWait    time.Duration
	NextIterWait time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		RndSeed:      time.Now().UnixNano(),
		RetryCount:   3,
		RetryWait:    10 * time.Second,
		NextIterWait: 6 * time.Hour,
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

	var it int
	for {
		it++
		select {
		case <-ctx.Done():
			return nil
		default:
			c.logger.Infof("iteration %d", it)
		}

		for _, addr := range addresses {
			node, err := findRandomNode(ctx, addr, cluster, opts.RndSeed)
			if err != nil {
				c.logger.Errorf("find node %s. Skipping. %w", addr.String(), err)
				continue
			}

			for i := 0; i <= opts.RetryCount; i++ {
				if i == opts.RetryCount {
					c.logger.Errorf("node %s: download for %s failed after %d tries", node.Name(), addr, opts.RetryCount)
					break
				}

				c.metrics.DownloadAttempts.Inc()
				c.logger.Infof("node %s: download attempt %d for %s", node.Name(), i+1, addr)

				start := time.Now()
				size, _, err := node.Client().DownloadFile(ctx, addr)
				if err != nil {
					c.metrics.DownloadErrors.Inc()
					c.logger.Errorf("node %s: download %s error: %v", node.Name(), addr, err)
					c.logger.Infof("retrying in: %v", opts.RetryWait)
					time.Sleep(opts.RetryWait)
					continue
				}
				c.logger.Infof("download size %d", size)
				c.metrics.DownloadSize.Set(float64(size))
				dur := time.Since(start)
				c.metrics.DownloadDuration.Observe(dur.Seconds())
				c.logger.Infof("node %s: downloaded %s successfully in %v", node.Name(), addr, dur)
				break
			}
		}

		c.logger.Infof("iteration %d completed", it)
		c.logger.Infof("sleeping for %v", opts.NextIterWait)
		time.Sleep(opts.NextIterWait)
	}
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

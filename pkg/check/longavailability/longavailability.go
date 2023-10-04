package longavailability

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
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

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o interface{}) (err error) {
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

	rnd := random.PseudoGenerator(opts.RndSeed)
	node, err := cluster.RandomNode(ctx, rnd)
	if err != nil {
		return fmt.Errorf("random node: %w", err)
	}
	client := node.Client()
	for _, addr := range addresses {
		t := &test{
			nodeName:   node.Name(),
			client:     client,
			logger:     c.logger,
			metrics:    c.metrics,
			retryCount: opts.RetryCount,
			retryWait:  opts.RetryWait,
		}
		err = t.run(ctx, addr)
		if err != nil {
			c.logger.Infof("node %s: download for %s failed: %v", node.Name(), addr, err)
		} else {
			c.logger.Infof("node %s: download for %s done", node.Name(), addr)
		}
	}
	return nil
}

type test struct {
	nodeName   string
	client     *bee.Client
	logger     logging.Logger
	metrics    metrics
	retryCount int
	retryWait  time.Duration
}

func (t *test) run(ctx context.Context, addr swarm.Address) error {
	for i := 0; i < t.retryCount; i++ {
		t.metrics.DownloadAttempts.Inc()
		t.logger.Infof("node %s: download attempt %d for %s", t.nodeName, i+1, addr)
		dur, err := t.download(ctx, addr)
		if err != nil {
			t.metrics.DownloadErrors.Inc()
			t.logger.Errorf("node %s: download %s error: %v", t.nodeName, addr, err)
			t.logger.Infof("retrying in: %v", t.retryWait)
			time.Sleep(t.retryWait)
			continue
		}
		t.metrics.DownloadDuration.Observe(dur.Seconds())
		t.logger.Infof("node %s: downloaded %s successfully in %v", t.nodeName, addr, dur)
		return nil
	}
	return fmt.Errorf("node %s: download %s failed after %d attempts", t.nodeName, addr, t.retryCount)
}

func (t *test) download(ctx context.Context, addr swarm.Address) (time.Duration, error) {
	start := time.Now()
	_, err := t.client.DownloadBytes(ctx, addr)
	if err != nil {
		return 0, fmt.Errorf("download from node %s: %w", t.nodeName, err)
	}
	dur := time.Since(start)
	return dur, nil
}

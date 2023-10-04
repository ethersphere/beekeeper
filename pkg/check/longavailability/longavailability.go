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

	for _, addr := range addresses {
		node, err := findRandomNode(ctx, addr, cluster, opts.RndSeed)
		if err != nil {
			return fmt.Errorf("find node: %w", err)
		}
		t := &test{
			node:       node,
			logger:     c.logger,
			metrics:    c.metrics,
			retryCount: opts.RetryCount,
			retryWait:  opts.RetryWait,
		}
		err = t.run(ctx, addr)
		if err != nil {
			c.logger.Errorf("node %s: download for %s failed: %v", node.Name(), addr, err)
		} else {
			c.logger.Infof("node %s: download for %s done", node.Name(), addr)
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

type test struct {
	node       orchestration.Node
	logger     logging.Logger
	metrics    metrics
	retryCount int
	retryWait  time.Duration
}

func (t *test) run(ctx context.Context, addr swarm.Address) error {
	for i := 0; i < t.retryCount; i++ {
		t.metrics.DownloadAttempts.Inc()
		t.logger.Infof("node %s: download attempt %d for %s", t.node.Name(), i+1, addr)
		dur, err := t.download(ctx, addr)
		if err != nil {
			t.metrics.DownloadErrors.Inc()
			t.logger.Errorf("node %s: download %s error: %v", t.node.Name(), addr, err)
			t.logger.Infof("retrying in: %v", t.retryWait)
			time.Sleep(t.retryWait)
			continue
		}
		t.metrics.DownloadDuration.Observe(dur.Seconds())
		t.logger.Infof("node %s: downloaded %s successfully in %v", t.node.Name(), addr, dur)
		return nil
	}
	return fmt.Errorf("node %s: download %s failed after %d attempts", t.node.Name(), addr, t.retryCount)
}

func (t *test) download(ctx context.Context, addr swarm.Address) (time.Duration, error) {
	start := time.Now()
	_, err := t.node.Client().DownloadBytes(ctx, addr)
	if err != nil {
		return 0, fmt.Errorf("download from node %s: %w", t.node.Name(), err)
	}
	dur := time.Since(start)
	return dur, nil
}

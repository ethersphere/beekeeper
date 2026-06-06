// Package radiusdecrease checks that puller workers recover promptly after a
// storage-radius decrease.
//
// When the reserve worker decreases the storage radius it calls manage(),
// which calls disconnectPeer() for every current peer.  If disconnectPeer()
// blocks while holding syncPeersMtx, manage() freezes for as long as the
// sync goroutines take to finish — indefinitely if the peers are still alive.
// This check verifies that manage() completes and workers restart within
// RecoveryTimeout after the radius decrease.
//
// # Required test binary
//
// This check MUST run against a bee binary built with the three
// radius-decrease CI patches (see bee/.github/patches/radius_decrease_*.patch):
//
//   - DefaultReserveCapacity = 200       (pkg/storer/storer.go)
//   - ReserveWakeUpDuration  = 10s       (pkg/storer/storer.go)
//   - threshold(capacity)    = capacity  (pkg/storer/reserve.go, 100 % not 50 %)
//
// Without these patches the reserve is 4.2 M chunks and a radius decrease
// cannot be triggered in CI time.
package radiusdecrease

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options holds tunable parameters for the check.
type Options struct {
	// UploadSizeMB is the total upload volume.  With 3 nodes and capacity=200
	// chunks, ≥4 MB guarantees at least one node overflows via pushsync routing.
	UploadSizeMB int
	// OverflowTimeout is the maximum time to wait for StorageRadius to reach 1
	// (reserve overflow confirmed) on any node.
	OverflowTimeout time.Duration
	// CascadeTimeout is the maximum time to wait for StorageRadius to fall back
	// to 0 (radius-decrease cascade triggered) after overflow.
	CascadeTimeout time.Duration
	// RecoveryTimeout is the maximum time to wait for PullsyncRate > 0 after
	// the radius decrease.  A timeout here indicates that manage() is blocked
	// and workers cannot restart.
	RecoveryTimeout time.Duration
	PostageLabel    string
	PostageAmount   int64
	PostageDepth    uint64
	Seed            int64
}

// NewDefaultOptions returns sensible defaults for CI.
func NewDefaultOptions() Options {
	return Options{
		UploadSizeMB:    4,
		OverflowTimeout: 5 * time.Minute,
		CascadeTimeout:  2 * time.Minute,
		RecoveryTimeout: 20 * time.Minute,
		PostageLabel:    "radius-decrease-check",
		PostageAmount:   1,
		PostageDepth:    17,
		Seed:            0,
	}
}

// compile-time interface check
var _ beekeeper.Action = (*Check)(nil)

// Check is the beekeeper action that tests puller recovery after radius decrease.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new Check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{logger: logger}
}

// Run executes the radius-decrease liveness check.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(o.Seed)

	uploadNode, err := cluster.RandomNode(ctx, rnd)
	if err != nil {
		return fmt.Errorf("random node: %w", err)
	}
	c.logger.Infof("upload node: %s", uploadNode.Name())

	// Flat list of all nodes for monitoring.
	allNodes := flatNodes(cluster)
	c.logger.Infof("monitoring %d nodes", len(allNodes))

	// Buy stamp and wait for usable (built-in poll inside CreatePostageBatch).
	batchID, err := uploadNode.Client().CreatePostageBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel, false)
	if err != nil {
		return fmt.Errorf("create postage batch: %w", err)
	}
	c.logger.Infof("postage batch ready: %s", batchID)

	// Pre-condition: all nodes must be at StorageRadius 0.
	for name, n := range allNodes {
		s, err := n.Client().Status(ctx)
		if err != nil {
			return fmt.Errorf("pre-check status node %s: %w", name, err)
		}
		if s.StorageRadius != 0 {
			return fmt.Errorf("pre-condition failed: node %s has StorageRadius %d, want 0", name, s.StorageRadius)
		}
	}

	// Seed the reserve with random bytes.  We upload in 512 KB blocks so we
	// don't allocate a single large buffer.  Content is simple XOR so each
	// block has a unique address.
	c.logger.Infof("uploading %d MB to seed reserve …", o.UploadSizeMB)
	totalBytes := o.UploadSizeMB * 1024 * 1024
	blockSize := 512 * 1024
	uploadOpts := api.UploadOptions{BatchID: batchID}
	for uploaded := 0; uploaded < totalBytes; uploaded += blockSize {
		size := blockSize
		if uploaded+blockSize > totalBytes {
			size = totalBytes - uploaded
		}
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(uploaded>>8) ^ byte(i)
		}
		if _, err := uploadNode.Client().UploadBytes(ctx, data, uploadOpts); err != nil {
			return fmt.Errorf("upload at offset %d: %w", uploaded, err)
		}
	}
	c.logger.Info("upload complete")

	// Phase 1: wait for StorageRadius to reach 1 on any node (overflow).
	c.logger.Info("waiting for reserve overflow (StorageRadius = 1) …")
	overflowNodeName, err := waitForRadius(ctx, allNodes, 1, o.OverflowTimeout)
	if err != nil {
		return fmt.Errorf("overflow phase: %w", err)
	}
	overflowNode := allNodes[overflowNodeName]
	c.logger.Infof("overflow on node %s", overflowNodeName)

	// Phase 2: wait for StorageRadius to fall back to 0 (radius decrease).
	// With ReserveWakeUpDuration=10s and threshold=100%, this fires within ~20s
	// once the pullsync rate drops to 0 after the initial sync burst.
	c.logger.Info("waiting for radius decrease (StorageRadius = 0) …")
	if _, err := waitForRadius(ctx, map[string]orchestration.Node{overflowNodeName: overflowNode}, 0, o.CascadeTimeout); err != nil {
		return fmt.Errorf("cascade phase: %w", err)
	}
	c.logger.Infof("radius decreased on node %s — disconnectPeer() called for all live peers", overflowNodeName)

	// Phase 3: measure worker recovery.  After the radius decrease, manage()
	// should reconnect peers and restart sync workers.  PullsyncRate > 0
	// confirms workers are running.  A timeout here means manage() is stuck.
	c.logger.Infof("waiting up to %s for PullsyncRate > 0 on node %s …", o.RecoveryTimeout, overflowNodeName)
	if err := waitForRecovery(ctx, overflowNodeName, overflowNode, o.RecoveryTimeout, c.logger); err != nil {
		return err
	}

	c.logger.Info("puller workers recovered after radius decrease — liveness confirmed")
	return nil
}

// flatNodes returns a name→Node map combining every node group in the cluster.
func flatNodes(cluster orchestration.Cluster) map[string]orchestration.Node {
	return cluster.Nodes()
}

// waitForRadius polls every node in the supplied map every 2 s until any
// node's StorageRadius equals target.  Returns the name of the first matching
// node, or an error if the timeout elapses.
func waitForRadius(ctx context.Context, nodes map[string]orchestration.Node, target uint8, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for name, n := range nodes {
			s, err := n.Client().Status(ctx)
			if err != nil {
				continue
			}
			if s.StorageRadius == target {
				return name, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return "", fmt.Errorf("timeout after %s waiting for StorageRadius = %d", timeout, target)
}

// waitForRecovery polls the named node every second until PullsyncRate > 0
// or the timeout elapses.  It logs progress every 10 s so the CI log shows
// the freeze duration rather than an apparent hang.
func waitForRecovery(ctx context.Context, name string, node orchestration.Node, timeout time.Duration, logger logging.Logger) error {
	deadline := time.Now().Add(timeout)
	start := time.Now()
	lastLog := time.Now()

	for time.Now().Before(deadline) {
		s, err := node.Client().Status(ctx)
		if err == nil && s.PullsyncRate > 0 {
			logger.Infof("node %s: PullsyncRate = %.4f after %s — workers recovered",
				name, s.PullsyncRate, time.Since(start).Round(time.Millisecond))
			return nil
		}

		if time.Since(lastLog) >= 10*time.Second {
			rate := 0.0
			if err == nil {
				rate = s.PullsyncRate
			}
			logger.Infof("node %s: waiting for recovery — PullsyncRate = %.4f, elapsed = %s",
				name, rate, time.Since(start).Round(time.Second))
			lastLog = time.Now()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return fmt.Errorf(
		"node %s: workers did not recover within %s after radius decrease "+
			"(PullsyncRate stayed at 0) — manage() goroutine may be blocked in disconnectPeer(); "+
			"see pkg/puller/puller.go",
		name, timeout,
	)
}

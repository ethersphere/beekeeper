// Package reserveradius stages a storage-radius change across a cluster and
// verifies the node recovers (pull-sync resumes) afterwards.
//
// It follows the empirical sequence found by the Phase-1 spike (see
// docs/radius-check-plan.md "Spike findings"):
//
//  1. wait for warmup/stabilization — the reserve worker's decrease loop is
//     gated on it, so a fresh node will not decrease for several minutes;
//  2. drive the radius UP by uploading until a node reaches the target;
//  3. let pushsync overshoot drain (uploads stopping != radius settling);
//  4. stop uploading and watch for a DECREASE and for pull-sync to recover
//     (pullsyncRate > 0) — a stuck puller manage()/disconnectPeer() shows as
//     no decrease / no recovery within the timeout (cf. beekeeper PR #581).
//
// This check requires a bee node patched for a small reserve (see
// bee/.github/patches/radius_*.patch); on stock capacity the radius never moves.
package reserveradius

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/test"
)

// Options represents reserve-radius check options.
type Options struct {
	RndSeed         int64
	PostageTTL      time.Duration
	PostageDepth    uint64
	PostageLabel    string
	UploadGroups    []string      // node groups to upload to / observe (empty = all full nodes)
	BlobSize        int64         // bytes per upload
	MaxUploads      int           // cap on uploads during the increase phase
	TargetRadius    uint8         // storageRadius to reach before stopping uploads
	WarmupWait      time.Duration // max wait for nodes to leave warmup before staging
	IncreaseTimeout time.Duration // max time to reach TargetRadius
	SettleWait      time.Duration // wait after uploads for pushsync overshoot to drain
	DecreaseTimeout time.Duration // max time to observe a decrease after uploads stop
	PollInterval    time.Duration
}

// NewDefaultOptions returns new default options.
func NewDefaultOptions() Options {
	return Options{
		RndSeed:         time.Now().UnixNano(),
		PostageTTL:      24 * time.Hour,
		PostageDepth:    22,
		PostageLabel:    "reserve-radius",
		UploadGroups:    []string{"bee"},
		BlobSize:        1 << 20, // 1 MiB
		MaxUploads:      60,
		TargetRadius:    1,
		WarmupWait:      15 * time.Minute,
		IncreaseTimeout: 5 * time.Minute,
		SettleWait:      time.Minute,
		DecreaseTimeout: 20 * time.Minute,
		PollInterval:    15 * time.Second,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns a new reserve-radius check.
func NewCheck(log logging.Logger) beekeeper.Action {
	return &Check{
		metrics: newMetrics("check_reserve_radius"),
		logger:  log,
	}
}

// Run stages a radius increase, then observes the decrease + pull-sync recovery.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return errors.New("invalid options type")
	}
	if o.TargetRadius == 0 {
		return errors.New("target-radius must be > 0")
	}

	c.logger.Infof("random seed: %d", o.RndSeed)
	rnd := random.PseudoGenerator(o.RndSeed)
	fullNodeClients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("get shuffled full node clients: %w", err)
	}
	nodes := fullNodeClients
	if len(o.UploadGroups) > 0 {
		nodes = fullNodeClients.FilterByNodeGroups(o.UploadGroups)
	}
	if len(nodes) < 1 {
		return fmt.Errorf("reserve-radius check requires at least 1 full node, got %d", len(nodes))
	}
	uploader := nodes[0] // a random node, since the list is shuffled
	c.logger.Infof("uploader: %s, observing %d node(s)", uploader.Name(), len(nodes))

	// 1. Wait for warmup/stabilization — the decrease loop is gated on it.
	if err := c.waitForWarmupDone(ctx, nodes, o); err != nil {
		return err
	}
	c.snapshot(ctx, nodes, "baseline")

	// 2. Drive the radius up by uploading.
	batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("create batch on %s: %w", uploader.Name(), err)
	}
	c.logger.WithField("batch_id", batchID).Infof("node %s: using batch", uploader.Name())
	peak := make(map[string]uint8, len(nodes))
	if err := c.driveIncrease(ctx, uploader, nodes, batchID, o, peak); err != nil {
		return err
	}

	// 3. Keep raising the per-node high-water `peak` through the settle window.
	//    The decrease can begin during settle (on an already-stabilised node it is
	//    near-immediate), so peak must be the max seen — a single post-settle read
	//    would come back at baseline and make a decrease impossible to detect.
	c.logger.Infof("uploads stopped; tracking peak for %s (pushsync settle)", o.SettleWait)
	settleDeadline := time.Now().Add(o.SettleWait)
	for time.Now().Before(settleDeadline) {
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err
		}
		c.updatePeak(ctx, nodes, peak)
	}
	c.logger.Infof("peak storageRadius per node: %v", peak)

	// 4. Observe the decrease + pull-sync recovery.
	return c.observeDecrease(ctx, nodes, peak, o)
}

// waitForWarmupDone blocks until every observed node reports isWarmingUp=false.
func (c *Check) waitForWarmupDone(ctx context.Context, nodes orchestration.ClientList, o Options) error {
	c.logger.Infof("waiting up to %s for nodes to finish warmup/stabilization", o.WarmupWait)
	deadline := time.Now().Add(o.WarmupWait)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		ready := true
		for _, n := range nodes {
			s, err := n.Status(ctx)
			if err != nil || s.IsWarmingUp {
				ready = false
			}
		}
		if ready {
			c.logger.Info("all observed nodes finished warmup")
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("nodes still warming up after %s", o.WarmupWait)
		}
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err
		}
	}
}

// driveIncrease uploads blobs to the uploader until any observed node reaches TargetRadius.
func (c *Check) driveIncrease(ctx context.Context, uploader *bee.Client, nodes orchestration.ClientList, batchID string, o Options, peak map[string]uint8) error {
	c.logger.Infof("driving increase: %d-byte blobs to %s until storageRadius>=%d (max %d uploads, timeout %s)",
		o.BlobSize, uploader.Name(), o.TargetRadius, o.MaxUploads, o.IncreaseTimeout)
	start := time.Now()
	deadline := start.Add(o.IncreaseTimeout)
	t := test.NewTest(c.logger)
	data := make([]byte, o.BlobSize)

	for i := 1; i <= o.MaxUploads; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("storageRadius did not reach %d within %s (%d uploads) — is the bee reserve patch active and is there enough data?", o.TargetRadius, o.IncreaseTimeout, i-1)
		}
		if _, err := crand.Read(data); err != nil {
			return fmt.Errorf("generate random data: %w", err)
		}
		if _, _, err := t.Upload(ctx, uploader, data, batchID, nil); err != nil {
			c.logger.Errorf("upload #%d failed: %v", i, err)
			continue
		}
		mx := c.updatePeak(ctx, nodes, peak)
		c.logger.Infof("increase: upload #%d, max storageRadius=%d (target %d)", i, mx, o.TargetRadius)
		if mx >= o.TargetRadius {
			c.metrics.TimeToIncrease.Set(time.Since(start).Seconds())
			c.logger.Infof("reached storageRadius %d after %d uploads (~%.1f MiB) in %s",
				mx, i, float64(int64(i)*o.BlobSize)/(1<<20), time.Since(start).Round(time.Second))
			return nil
		}
	}
	return fmt.Errorf("storageRadius did not reach %d after %d uploads", o.TargetRadius, o.MaxUploads)
}

// observeDecrease watches for any node's radius to fall below its peak, and for
// pull-sync to recover (pullsyncRate>0). No decrease within the timeout is the
// failure signal (cf. PR #581 puller manage()/disconnectPeer() stall).
func (c *Check) observeDecrease(ctx context.Context, nodes orchestration.ClientList, peak map[string]uint8, o Options) error {
	c.logger.Infof("watching for radius decrease + pull-sync recovery (timeout %s)", o.DecreaseTimeout)
	start := time.Now()
	deadline := start.Add(o.DecreaseTimeout)
	recovered := false

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		decreasedOn := ""
		for _, n := range nodes {
			s, err := n.Status(ctx)
			if err != nil {
				continue
			}
			c.emit(n.Name(), s)
			if s.PullsyncRate > 0 {
				recovered = true
			}
			if p, ok := peak[n.Name()]; ok && s.StorageRadius < p {
				decreasedOn = n.Name()
			}
		}
		if decreasedOn != "" {
			c.metrics.TimeToDecrease.Set(time.Since(start).Seconds())
			c.logger.Infof("radius decrease observed on %s after %s (pull-sync recovery seen: %t)",
				decreasedOn, time.Since(start).Round(time.Second), recovered)
			// TODO(reserve-radius): make pull-sync recovery a hard gate once the
			// expected rate floor is characterised on a data-heavy cluster (Phase 3).
			if !recovered {
				c.logger.Warning("decrease observed but pullsyncRate stayed 0 — puller may not have resumed; investigate (cf. PR #581)")
			}
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no radius decrease within %s after uploads stopped — puller manage()/disconnectPeer() may be stalled (cf. PR #581), or the decrease gate (synced + count<threshold) was not met", o.DecreaseTimeout)
		}
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err
		}
	}
}

// snapshot polls every node, emits metrics, logs a line, and returns the max storageRadius.
func (c *Check) snapshot(ctx context.Context, nodes orchestration.ClientList, phase string) uint8 {
	var mx uint8
	for _, n := range nodes {
		s, err := n.Status(ctx)
		if err != nil {
			c.logger.Debugf("%s: status error: %v", n.Name(), err)
			continue
		}
		c.emit(n.Name(), s)
		if s.StorageRadius > mx {
			mx = s.StorageRadius
		}
		c.logger.Infof("[%s] %s: storageRadius=%d reserveSize=%d withinR=%d pullsyncRate=%.4f warmingUp=%t",
			phase, n.Name(), s.StorageRadius, s.ReserveSize, s.ReserveSizeWithinRadius, s.PullsyncRate, s.IsWarmingUp)
	}
	return mx
}

// updatePeak polls each node, emits metrics, raises the per-node high-water peak,
// logs a line, and returns the current max storageRadius across nodes.
func (c *Check) updatePeak(ctx context.Context, nodes orchestration.ClientList, peak map[string]uint8) uint8 {
	var mx uint8
	for _, n := range nodes {
		s, err := n.Status(ctx)
		if err != nil {
			c.logger.Debugf("%s: status error: %v", n.Name(), err)
			continue
		}
		c.emit(n.Name(), s)
		if s.StorageRadius > peak[n.Name()] {
			peak[n.Name()] = s.StorageRadius
		}
		if s.StorageRadius > mx {
			mx = s.StorageRadius
		}
		c.logger.Infof("%s: storageRadius=%d (peak %d) reserveSize=%d withinR=%d pullsyncRate=%.4f",
			n.Name(), s.StorageRadius, peak[n.Name()], s.ReserveSize, s.ReserveSizeWithinRadius, s.PullsyncRate)
	}
	return mx
}

func (c *Check) emit(node string, s *api.StatusResponse) {
	c.metrics.StorageRadius.WithLabelValues(node).Set(float64(s.StorageRadius))
	c.metrics.ReserveSize.WithLabelValues(node).Set(float64(s.ReserveSize))
	c.metrics.PullsyncRate.WithLabelValues(node).Set(s.PullsyncRate)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

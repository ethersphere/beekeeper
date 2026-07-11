// Package reserveradius drives a storage-radius change on a Bee cluster and verifies the
// radius comes back down once pull-sync settles. It runs a single self-driving flow:
//
//  1. wait for warmup/stabilization — the reserve worker gates its decrease loop on it;
//  2. drive the radius UP by uploading random data to random full nodes until a target is reached;
//  3. stop uploading and wait for pull-sync to go idle (pullsyncRate==0) — the reserve worker
//     only decreases the radius while SyncRate()==0 (== /status pullsyncRate), so the whole
//     neighbourhood must finish syncing first;
//  4. observe the radius tick back down below its peak (the reserve worker's decrease), failing
//     if no decrease appears within the timeout — a stuck puller shows as no decrease.
//
// This check requires a bee node patched for a small reserve (see
// bee/.github/patches/radius_*.patch); on stock capacity the radius never moves.
package reserveradius

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
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
	PostageLabel    string        // this check's own batch label
	UploadGroups    []string      // node groups to upload to / observe (empty = all full nodes)
	BlobSize        int64         // bytes per upload
	MaxUploads      int           // cap on uploads during the increase phase
	TargetRadius    uint8         // storageRadius any node must reach before stopping uploads
	ForceDecrease   bool          // after pull-sync idles, dilute a batch to expiry to force the decrease (vs waiting for the idle tick)
	DiluteStep      uint64        // depth increase applied per dilution (each step ~halves remaining TTL)
	MaxDilutions    int           // cap on dilutions per batch when forcing expiry
	GasPrice        string        // gas price for the dilute tx (empty = node default)
	WarmupWait      time.Duration // max wait for nodes to leave warmup before driving
	IncreaseTimeout time.Duration // max time to reach TargetRadius
	SyncSettleWait  time.Duration // after uploads stop, max wait for pullsyncRate==0 on all nodes (the decrease gate)
	DecreaseTimeout time.Duration // max time to observe a radius decrease after pull-sync idles
	PollInterval    time.Duration
}

// NewDefaultOptions returns new default options.
func NewDefaultOptions() Options {
	return Options{
		RndSeed:         time.Now().UnixNano(),
		PostageTTL:      48 * time.Hour,
		PostageDepth:    22,
		PostageLabel:    "reserve-radius",
		UploadGroups:    []string{"bee"},
		BlobSize:        8 << 20, // 8 MiB
		MaxUploads:      200,
		TargetRadius:    3,
		ForceDecrease:   true,
		DiluteStep:      1, // depth += 1 per dilution ~halves remaining TTL; larger jumps revert once they cross the min-validity floor
		MaxDilutions:    20,
		GasPrice:        "",
		WarmupWait:      15 * time.Minute,
		IncreaseTimeout: 30 * time.Minute,
		SyncSettleWait:  10 * time.Minute,
		DecreaseTimeout: 6 * time.Hour, // testnet: batch expiry lands ~minimumValidityBlocks (~5h) after diluting to the floor
		PollInterval:    15 * time.Second,
	}
}

// pullSyncIdleRate is the pullsyncRate at or below which pull-sync is treated as idle.
// The rate is a decaying windowed average that settles to a small residual rather than
// exactly 0, so an == 0 gate would never open; active sync is orders of magnitude higher.
const pullSyncIdleRate = 0.05

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

// batchRef is a postage batch created during the increase phase, paired with the node it lives on.
type batchRef struct {
	node    *bee.Client
	batchID string
}

// Run drives the radius up by uploading random data to random nodes, waits for pull-sync to
// go idle (the reserve worker's decrease gate), optionally forces the decrease by diluting a
// batch to expiry, then verifies the radius ticks back down.
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
	nodes, err := c.selectNodes(ctx, cluster, rnd, o)
	if err != nil {
		return err
	}
	c.logger.Infof("observing %d node(s); uploading random data to random nodes under label %q", len(nodes), o.PostageLabel)

	// 1. Wait for warmup/stabilization — the decrease loop is gated on it.
	if err := c.waitForWarmupDone(ctx, nodes, o); err != nil {
		return err
	}
	c.snapshot(ctx, nodes, "baseline")

	// 2. Drive the radius up by uploading to random nodes; peak tracks the per-node high-water.
	peak := make(map[string]uint8, len(nodes))
	batches, err := c.driveIncrease(ctx, nodes, rnd, peak, o)
	if err != nil {
		return err
	}

	// 3. Stop uploading and wait for pull-sync to go idle. The reserve worker only decreases
	//    the radius when SyncRate()==0 (== /status pullsyncRate), so the neighbourhood must
	//    finish syncing first. Keep raising peak in case a decrease begins during the wait.
	c.waitPullSyncIdle(ctx, nodes, peak, o)

	// 4. Optionally hasten the decrease: dilute one batch down to the contract's minimum
	//    validity so it expires within the check's budget (~minimumValidityBlocks, ~hours)
	//    instead of its full TTL. On expiry its chunks are evicted and reserve commitment
	//    drops, which — with idle pull-sync — lets the reserve worker decrease the radius.
	//    Diluting a single batch is a partial expiry: it drops commitment without evicting
	//    the whole reserve (which would halt the neighbourhood).
	if o.ForceDecrease && len(batches) > 0 {
		c.diluteToMinValidity(ctx, batches[0], o)
	}

	// 5. Observe the radius tick back down below its peak (via the idle tick, or batch expiry).
	return c.observeDecrease(ctx, nodes, peak, o)
}

// selectNodes returns the observed full-node clients (shuffled, optionally filtered to UploadGroups).
func (c *Check) selectNodes(ctx context.Context, cluster orchestration.Cluster, rnd *rand.Rand, o Options) (orchestration.ClientList, error) {
	fullNodeClients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return nil, fmt.Errorf("get shuffled full node clients: %w", err)
	}
	nodes := fullNodeClients
	if len(o.UploadGroups) > 0 {
		nodes = fullNodeClients.FilterByNodeGroups(o.UploadGroups)
	}
	if len(nodes) < 1 {
		return nil, fmt.Errorf("reserve-radius check requires at least 1 full node, got %d", len(nodes))
	}
	return nodes, nil
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

// driveIncrease uploads BlobSize random blobs to randomly-chosen nodes until any observed
// node reaches TargetRadius. Each node uploads under its own mutable batch (this check's
// label). It raises the per-node high-water peak as it goes and returns the batches it
// created (in creation order) for the optional force-decrease step.
func (c *Check) driveIncrease(ctx context.Context, nodes orchestration.ClientList, rnd *rand.Rand, peak map[string]uint8, o Options) ([]batchRef, error) {
	c.logger.Infof("driving increase: %d-byte blobs to random nodes until storageRadius>=%d (max %d uploads, timeout %s)",
		o.BlobSize, o.TargetRadius, o.MaxUploads, o.IncreaseTimeout)
	start := time.Now()
	deadline := start.Add(o.IncreaseTimeout)
	t := test.NewTest(c.logger)
	data := make([]byte, o.BlobSize)
	batchByNode := make(map[string]string, len(nodes))
	var batches []batchRef

	for i := 1; i <= o.MaxUploads; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("storageRadius did not reach %d within %s (%d uploads) — is the bee reserve patch active and is there enough data?", o.TargetRadius, o.IncreaseTimeout, i-1)
		}

		uploader := nodes[rnd.Intn(len(nodes))]
		batchID, ok := batchByNode[uploader.Name()]
		if !ok {
			id, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
			if err != nil {
				// Transient per-node error (e.g. a 503 from chainstate); another random
				// node is picked next iteration, so skip rather than fail the whole run.
				c.logger.Warningf("create batch on %s failed, skipping upload #%d: %v", uploader.Name(), i, err)
				continue
			}
			batchID = id
			batchByNode[uploader.Name()] = id
			batches = append(batches, batchRef{node: uploader, batchID: id})
			c.logger.WithField("batch_id", id).Infof("node %s: using batch", uploader.Name())
		}

		if _, err := crand.Read(data); err != nil {
			return nil, fmt.Errorf("generate random data: %w", err)
		}
		if _, _, err := t.Upload(ctx, uploader, data, batchID, nil); err != nil {
			c.logger.Errorf("upload #%d to %s failed: %v", i, uploader.Name(), err)
			continue
		}

		_, mx := c.updatePeak(ctx, nodes, peak)
		c.logger.Infof("increase: upload #%d to %s, max storageRadius=%d (target %d)", i, uploader.Name(), mx, o.TargetRadius)
		if mx >= o.TargetRadius {
			c.metrics.TimeToIncrease.Set(time.Since(start).Seconds())
			c.logger.Infof("reached storageRadius %d after %d uploads (~%.1f MiB) across %d batch(es) in %s",
				mx, i, float64(int64(i)*o.BlobSize)/(1<<20), len(batches), time.Since(start).Round(time.Second))
			return batches, nil
		}
	}
	return nil, fmt.Errorf("storageRadius did not reach %d after %d uploads", o.TargetRadius, o.MaxUploads)
}

// diluteToMinValidity dilutes a batch as far down as the contract allows — each +DiluteStep
// (default 1) roughly halves its remaining TTL — stopping when a dilution is rejected because
// it would drop the batch's validity below the chain's minimumValidityBlocks floor. That floor
// makes expiry itself unreachable by dilution, so this only *hastens* the natural expiry: it
// leaves the batch at its minimum TTL (~minimumValidityBlocks) so it expires within the check's
// budget rather than its full original TTL. On expiry the batch's chunks are evicted and reserve
// commitment drops. Best-effort — a read/dilute error just stops early (the batch keeps whatever
// TTL it reached), so this never fails the check on its own.
func (c *Check) diluteToMinValidity(ctx context.Context, b batchRef, o Options) {
	c.logger.Infof("diluting batch %s on %s toward minimum validity to hasten expiry", b.batchID, b.node.Name())
	for k := 0; k < o.MaxDilutions; k++ {
		if ctx.Err() != nil {
			return
		}
		stamp, err := b.node.PostageStamp(ctx, b.batchID)
		if err != nil {
			c.logger.Warningf("read batch %s on %s: %v; stopping dilution", b.batchID, b.node.Name(), err)
			return
		}
		if !stamp.Exists || stamp.BatchTTL <= 0 {
			c.logger.Infof("batch %s on %s already expired (ttl=%ds)", b.batchID, b.node.Name(), stamp.BatchTTL)
			return
		}
		newDepth := uint64(stamp.Depth) + o.DiluteStep
		if err := b.node.DilutePostageBatch(ctx, b.batchID, newDepth, o.GasPrice); err != nil {
			// Rejected: the resulting validity would fall below minimumValidityBlocks — the
			// batch is at its minimum TTL, the expected stop condition.
			c.logger.Infof("batch %s on %s at minimum validity: ttl ~%ds (~%.1fh); dilute to depth %d rejected (%v)",
				b.batchID, b.node.Name(), stamp.BatchTTL, float64(stamp.BatchTTL)/3600, newDepth, err)
			return
		}
		c.metrics.Dilutions.Inc()
		c.logger.Infof("diluted batch %s on %s to depth %d (ttl was %ds ~%.1fh)", b.batchID, b.node.Name(), newDepth, stamp.BatchTTL, float64(stamp.BatchTTL)/3600)
	}
	c.logger.Warningf("batch %s on %s: reached max %d dilutions without hitting the validity floor", b.batchID, b.node.Name(), o.MaxDilutions)
}

// waitPullSyncIdle blocks until every node reports pullsyncRate==0 (pull-sync idle) or
// SyncSettleWait elapses. Idle is the reserve worker's decrease precondition, so this is a
// best-effort gate: on timeout it logs and proceeds (observeDecrease will fail if the radius
// never drops). It keeps raising peak, since a decrease can begin as soon as sync idles.
func (c *Check) waitPullSyncIdle(ctx context.Context, nodes orchestration.ClientList, peak map[string]uint8, o Options) {
	c.logger.Infof("uploads stopped; waiting up to %s for pull-sync to go idle (pullsyncRate<=%.2f on all %d node(s))", o.SyncSettleWait, pullSyncIdleRate, len(nodes))
	deadline := time.Now().Add(o.SyncSettleWait)
	for {
		idle, maxRate := true, 0.0
		for _, n := range nodes {
			s, err := n.Status(ctx)
			if err != nil {
				idle = false
				continue
			}
			c.emit(n.Name(), s)
			if s.StorageRadius > peak[n.Name()] {
				peak[n.Name()] = s.StorageRadius
			}
			if s.PullsyncRate > pullSyncIdleRate {
				idle = false
			}
			if s.PullsyncRate > maxRate {
				maxRate = s.PullsyncRate
			}
		}
		if idle {
			c.logger.Infof("pull-sync idle on all nodes (pullsyncRate<=%.2f) — decrease gate open", pullSyncIdleRate)
			return
		}
		if time.Now().After(deadline) {
			c.logger.Warningf("pull-sync did not go idle within %s (max pullsyncRate=%.4f); proceeding to observe anyway", o.SyncSettleWait, maxRate)
			return
		}
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return // context cancelled (e.g. check timeout) — stop waiting
		}
	}
}

// observeDecrease watches for any node's storageRadius to fall below its peak — the reserve
// worker's decrease once pull-sync is idle. No decrease within DecreaseTimeout is the failure
// signal (a stuck puller never lets SyncRate reach 0, so the radius never decreases).
func (c *Check) observeDecrease(ctx context.Context, nodes orchestration.ClientList, peak map[string]uint8, o Options) error {
	c.logger.Infof("watching for a radius decrease below the peak %v (timeout %s)", peak, o.DecreaseTimeout)
	start := time.Now()
	deadline := start.Add(o.DecreaseTimeout)
	for {
		for _, n := range nodes {
			s, err := n.Status(ctx)
			if err != nil {
				continue
			}
			c.emit(n.Name(), s)
			if s.StorageRadius > peak[n.Name()] {
				peak[n.Name()] = s.StorageRadius
			}
			if s.StorageRadius < peak[n.Name()] {
				c.metrics.TimeToDecrease.Set(time.Since(start).Seconds())
				c.logger.Infof("radius decrease observed on %s (%d -> %d) after %s — done",
					n.Name(), peak[n.Name()], s.StorageRadius, time.Since(start).Round(time.Second))
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no radius decrease below peak %v within %s after pull-sync idled — the reserve worker never ticked the radius down (a stuck puller keeps SyncRate>0, cf. PR #581)", peak, o.DecreaseTimeout)
		}
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err
		}
	}
}

// updatePeak polls each node, emits metrics, raises the per-node high-water peak, and
// returns the min and max current storageRadius across the nodes that responded.
func (c *Check) updatePeak(ctx context.Context, nodes orchestration.ClientList, peak map[string]uint8) (minR, maxR uint8) {
	seen := false
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
		if s.StorageRadius > maxR {
			maxR = s.StorageRadius
		}
		if !seen || s.StorageRadius < minR {
			minR = s.StorageRadius
			seen = true
		}
	}
	return minR, maxR
}

// snapshot polls every node, emits metrics, and logs a per-node line for the given phase.
func (c *Check) snapshot(ctx context.Context, nodes orchestration.ClientList, phase string) {
	for _, n := range nodes {
		s, err := n.Status(ctx)
		if err != nil {
			c.logger.Debugf("%s: status error: %v", n.Name(), err)
			continue
		}
		c.emit(n.Name(), s)
		c.logger.Infof("[%s] %s: storageRadius=%d reserveSize=%d withinR=%d pullsyncRate=%.4f warmingUp=%t",
			phase, n.Name(), s.StorageRadius, s.ReserveSize, s.ReserveSizeWithinRadius, s.PullsyncRate, s.IsWarmingUp)
	}
}

func (c *Check) emit(node string, s *api.StatusResponse) {
	c.metrics.StorageRadius.WithLabelValues(node).Set(float64(s.StorageRadius))
	c.metrics.ReserveSize.WithLabelValues(node).Set(float64(s.ReserveSize))
	c.metrics.ReserveWithinRadius.WithLabelValues(node).Set(float64(s.ReserveSizeWithinRadius))
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

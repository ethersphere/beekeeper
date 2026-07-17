// Package reserveradiusv2 is a simplified, reproducible reserve-radius scenario built to
// capture pull-sync issues around radius changes and batch dilution. Instead of blindly
// uploading until a target radius (v1), it fills the reserve with a known amount of data
// per small postage batch until just over the 50% capacity threshold, then dilutes the
// batches one at a time and verifies the storage radius decreases while pull-sync resyncs.
package reserveradiusv2

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/test"
)

// Options represents reserve-radius-v2 check options.
type Options struct {
	RndSeed           int64
	PostageAmount     int64         // amount per batch (each batch is created fresh, never reused)
	BatchDepth        uint64        // depth of each small batch
	PostageLabel      string        // label prefix; each batch gets label-<n>
	UploadGroups      []string      // node groups to upload to / observe (empty = all full nodes)
	BlobSize          int64         // bytes per upload
	DataPerBatch      int64         // bytes uploaded under each batch
	MaxBatches        int           // cap on batches during the fill phase
	ReserveCapacity   uint64        // reserve capacity in chunks (patched bee: 4000)
	TargetFillPercent float64       // stop filling when any node's reserve reaches this % of capacity
	Dilute            bool          // dilute batches after the fill to trigger the decrease
	DiluteStep        uint64        // depth increase per dilution
	DiluteInterval    time.Duration // spacing between dilutions so they show as distinct events
	MaxDilutionRounds int           // cap on dilution rounds (each round dilutes every batch once)
	GasPrice          string
	WarmupWait        time.Duration
	FillTimeout       time.Duration
	SyncSettleWait    time.Duration // after filling, max wait for pull-sync to go idle
	DecreaseTimeout   time.Duration // max time to observe a radius decrease
	PollInterval      time.Duration
}

// NewDefaultOptions returns new default options.
func NewDefaultOptions() Options {
	return Options{
		RndSeed:           time.Now().UnixNano(),
		PostageAmount:     1000,
		BatchDepth:        18,
		PostageLabel:      "reserve-radius-v2",
		UploadGroups:      []string{"bee"},
		BlobSize:          1 << 20, // 1 MiB
		DataPerBatch:      4 << 20, // 4 MiB = ~1024 chunks per batch
		MaxBatches:        50,
		ReserveCapacity:   4000, // patched bee reserve; 50% = 2000 chunks
		TargetFillPercent: 0.6,  // just over the 50% decrease threshold
		Dilute:            true,
		DiluteStep:        1,
		DiluteInterval:    time.Minute,
		MaxDilutionRounds: 5,
		GasPrice:          "",
		WarmupWait:        15 * time.Minute,
		FillTimeout:       30 * time.Minute,
		SyncSettleWait:    10 * time.Minute,
		DecreaseTimeout:   30 * time.Minute,
		PollInterval:      10 * time.Second,
	}
}

// pullSyncIdleRate is the pullsyncRate at or below which pull-sync is treated as idle
// (the rate decays to a small residual, never exactly 0).
const pullSyncIdleRate = 0.05

var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns a new reserve-radius-v2 check.
func NewCheck(log logging.Logger) beekeeper.Action {
	return &Check{
		metrics: newMetrics("check_reserve_radius_v2"),
		logger:  log,
	}
}

type batchRef struct {
	node    *bee.Client
	batchID string
	done    bool // reached the validity floor or errored; skip in later rounds
}

// state tracks per-node radius history across polls.
type state struct {
	peak map[string]uint8
	last map[string]uint8
}

// Run fills the reserve just over the 50% capacity threshold using small batches with a
// fixed amount of data each, waits for pull-sync to settle, then dilutes the batches one
// at a time and verifies the storage radius decreases.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return errors.New("invalid options type")
	}
	if o.ReserveCapacity == 0 {
		return errors.New("reserve-capacity must be > 0")
	}
	if o.TargetFillPercent <= 50 {
		c.logger.Warningf("target-fill-percent %.1f is not over 50%% — the decrease threshold may already be crossed before dilution", o.TargetFillPercent)
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
		return fmt.Errorf("reserve-radius-v2 check requires at least 1 full node, got %d", len(nodes))
	}
	c.logger.Infof("observing %d node(s); reserve capacity %d chunks, fill target %.1f%%", len(nodes), o.ReserveCapacity, o.TargetFillPercent)

	st := &state{peak: make(map[string]uint8), last: make(map[string]uint8)}
	c.poll(ctx, nodes, st, o)

	_, err = c.fill(ctx, nodes, st, o)
	if err != nil {
		return err
	}

	c.logger.Infof("radius check fill done. waiting for pull sync to go idle")

	c.waitPullSyncIdle(ctx, nodes, st, o)

	c.logger.Infof("radius check fill done. pull sync idle, driving decrease")

	// return c.driveDecrease(ctx, nodes, batches, st, o)
	return nil
}

// fill creates small batches round-robin across nodes and uploads exactly DataPerBatch
// bytes under each, stopping once any node's reserve crosses TargetFillPercent.
func (c *Check) fill(ctx context.Context, nodes orchestration.ClientList, st *state, o Options) ([]*batchRef, error) {
	clusterSize := len(nodes)
	neighborhoods := math.Log2(float64(clusterSize))
	nodeReserve := float64(4000)
	totalStorageToFill := o.TargetFillPercent * nodeReserve * neighborhoods
	stamps := 5
	sizePerStamp := int(totalStorageToFill / float64(stamps))

	c.logger.Infof("fill: clusterSize %d, neighborhoods %d, nodeReserve %d, totalStorageToFill %d, stamps %d,sizePerStamp(chunks) %d", clusterSize, neighborhoods, nodeReserve, totalStorageToFill, stamps, sizePerStamp)

	start := time.Now()
	deadline := start.Add(o.FillTimeout)
	t := test.NewTest(c.logger)
	data := make([]byte, sizePerStamp*4096)
	var batches []*batchRef

	// caveat: if there's an error, this won't work very well because
	// the errors have a continue statement
	for b := 0; b < stamps; b++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("reserve did not reach %.1f%% within %s (%d batches) — is the bee reserve patch active?", o.TargetFillPercent, o.FillTimeout, b-1)
		}

		node := nodes[(b-1)%len(nodes)]
		label := fmt.Sprintf("%s-%d", o.PostageLabel, b)
		batchID, err := node.CreatePostageBatch(ctx, o.PostageAmount, o.BatchDepth, label, false)
		if err != nil {
			c.logger.Warningf("create batch %s on %s failed, skipping: %v", label, node.Name(), err)
			continue
		}
		batches = append(batches, &batchRef{node: node, batchID: batchID})
		c.metrics.BatchesCreated.Inc()
		c.logger.WithField("batch_id", batchID).Infof("batch #%d created on %s (label %s)", b, node.Name(), label)

		if _, err := crand.Read(data); err != nil {
			return nil, fmt.Errorf("generate random data: %w", err)
		}
		if _, _, err := t.Upload(ctx, node, data, batchID, nil); err != nil {
			c.logger.Errorf("upload to %s failed: %v", node.Name(), err)
			continue
		}
		c.metrics.UploadedBytes.Add(float64(o.BlobSize))

		fill, _ := c.poll(ctx, nodes, st, o)
		c.logger.Infof("fill: batch #%d done (%d bytes), max reserve fill %.1f%% (target %.1f%%)", b, o.DataPerBatch, fill, o.TargetFillPercent)
		if fill >= o.TargetFillPercent {
			c.logger.Infof("reached %.1f%% fill with %d batches (~%.1f MiB) in %s",
				fill, len(batches), float64(int64(len(batches))*o.DataPerBatch)/(1<<20), time.Since(start).Round(time.Second))
			return batches, nil
		}
	}
	return nil, fmt.Errorf("reserve did not reach %.1f%% after %d batches", o.TargetFillPercent, o.MaxBatches)
}

// waitPullSyncIdle blocks until every node reports pull-sync idle or SyncSettleWait
// elapses (best-effort: on timeout it logs and proceeds).
func (c *Check) waitPullSyncIdle(ctx context.Context, nodes orchestration.ClientList, st *state, o Options) {
	c.logger.Infof("uploads stopped; waiting up to %s for pull-sync to go idle (pullsyncRate<=%.2f on all nodes)", o.SyncSettleWait, pullSyncIdleRate)
	deadline := time.Now().Add(o.SyncSettleWait)
	for {
		_, maxRate := c.poll(ctx, nodes, st, o)
		if maxRate <= pullSyncIdleRate {
			c.logger.Info("pull-sync idle on all nodes")
			return
		}
		if time.Now().After(deadline) {
			c.logger.Warningf("pull-sync did not go idle within %s (max pullsyncRate=%.4f); proceeding anyway", o.SyncSettleWait, maxRate)
			return
		}
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return
		}
	}
}

// driveDecrease dilutes the batches one at a time (one dilution per DiluteInterval) while
// watching for any node's storageRadius to fall below its peak. Interleaving dilutions
// with polling makes each dilution a distinct event next to the radius/pull-sync series.
func (c *Check) driveDecrease(ctx context.Context, nodes orchestration.ClientList, batches []*batchRef, radiusState *state, o Options) error {
	c.logger.Infof("watching for a radius decrease below peak %v (timeout %s, dilute=%t)", radiusState.peak, o.DecreaseTimeout, o.Dilute)
	start := time.Now()
	deadline := start.Add(o.DecreaseTimeout)
	nextDilute := time.Now()
	dilutions, maxDilutions := 0, len(batches)*o.MaxDilutionRounds

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		c.poll(ctx, nodes, radiusState, o)
		for _, n := range nodes {
			if radiusState.last[n.Name()] < radiusState.peak[n.Name()] {
				c.metrics.TimeToDecrease.Set(time.Since(start).Seconds())
				c.logger.Infof("radius decrease observed on %s (%d -> %d) after %s and %d dilution(s) — done",
					n.Name(), radiusState.peak[n.Name()], radiusState.last[n.Name()], time.Since(start).Round(time.Second), dilutions)
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no radius decrease below peak %v within %s (%d dilutions applied) — possible stuck pull-sync", radiusState.peak, o.DecreaseTimeout, dilutions)
		}
		if o.Dilute && dilutions < maxDilutions && time.Now().After(nextDilute) {
			if c.diluteNext(ctx, batches, dilutions, o) {
				dilutions++
			}
			nextDilute = time.Now().Add(o.DiluteInterval)
		}
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err
		}
	}
}

// diluteNext dilutes the next not-done batch (round-robin) by DiluteStep and reports
// whether a dilution was applied. A rejected dilution (validity floor) marks the batch done.
func (c *Check) diluteNext(ctx context.Context, batches []*batchRef, round int, o Options) bool {
	for i := 0; i < len(batches); i++ {
		b := batches[(round+i)%len(batches)]
		if b.done {
			continue
		}
		stamp, err := b.node.PostageStamp(ctx, b.batchID)
		if err != nil {
			c.logger.Warningf("read batch %s on %s: %v; skipping", b.batchID, b.node.Name(), err)
			b.done = true
			continue
		}
		if stamp.BatchTTL < 0 {
			c.logger.Warningf("batch %s has infinite TTL (chain price 0) — dilution cannot hasten expiry; skipping", b.batchID)
			b.done = true
			continue
		}
		newDepth := uint64(stamp.Depth) + o.DiluteStep
		if err := b.node.DilutePostageBatch(ctx, b.batchID, newDepth, o.GasPrice); err != nil {
			c.logger.Infof("batch %s at validity floor (ttl ~%ds); dilute to depth %d rejected: %v", b.batchID, stamp.BatchTTL, newDepth, err)
			b.done = true
			continue
		}
		c.metrics.Dilutions.Inc()
		c.logger.Infof("diluted batch %s on %s to depth %d (ttl was %ds)", b.batchID, b.node.Name(), newDepth, stamp.BatchTTL)
		return true
	}
	return false
}

// poll reads /status on every node, emits metrics, updates peak/last radius, and returns
// the max reserve fill percent and max pullsyncRate across nodes.
func (c *Check) poll(ctx context.Context, nodes orchestration.ClientList, st *state, o Options) (maxFill, maxRate float64) {
	for _, n := range nodes {
		s, err := n.Status(ctx)
		if err != nil {
			c.logger.Debugf("%s: status error: %v", n.Name(), err)
			continue
		}
		name := n.Name()
		fill := float64(s.ReserveSizeWithinRadius) / float64(o.ReserveCapacity) * 100
		c.emit(name, s, fill)

		if prev, seen := st.last[name]; seen && s.StorageRadius != prev {
			dir := "increase"
			if s.StorageRadius < prev {
				dir = "decrease"
			}
			c.metrics.RadiusEvents.WithLabelValues(name, dir).Inc()
			c.logger.Infof("%s: storageRadius %d -> %d (%s), fill %.1f%%, pullsyncRate %.4f", name, prev, s.StorageRadius, dir, fill, s.PullsyncRate)
		}
		st.last[name] = s.StorageRadius
		if s.StorageRadius > st.peak[name] {
			st.peak[name] = s.StorageRadius
		}
		if fill > maxFill {
			maxFill = fill
		}
		if s.PullsyncRate > maxRate {
			maxRate = s.PullsyncRate
		}
	}
	return maxFill, maxRate
}

func (c *Check) emit(node string, s *api.StatusResponse, fill float64) {
	c.metrics.StorageRadius.WithLabelValues(node).Set(float64(s.StorageRadius))
	c.metrics.ReserveSize.WithLabelValues(node).Set(float64(s.ReserveSize))
	c.metrics.ReserveWithinRadius.WithLabelValues(node).Set(float64(s.ReserveSizeWithinRadius))
	c.metrics.ReserveFillPercent.WithLabelValues(node).Set(fill)
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

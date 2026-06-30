// Package reserveradius stages a storage-radius change across a cluster and
// verifies the node recovers (pull-sync resumes) afterwards.
//
// It follows the empirical sequence found by the Phase-1 spike (see
// docs/radius-check-plan.md "Spike findings"):
//
//  1. wait for warmup/stabilization — the reserve worker's decrease loop is
//     gated on it, so a fresh node will not decrease for several minutes;
//  2. drive the radius UP by uploading until ALL observed nodes reach the target
//     committedDepth (confirmed over consecutive polls, not just the uploader);
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
	"math"
	"math/big"
	"strings"
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
	Mode              string        // "drive" (upload to force a change) or "observe" (monitor a change driven externally)
	Duration          time.Duration // observe mode: total monitor run length
	RndSeed           int64
	PostageTTL        time.Duration
	PostageDepth      uint64
	PostageLabel      string
	UploadGroups      []string      // node groups to upload to / observe (empty = all full nodes)
	BlobSize          int64         // bytes per upload
	MaxUploads        int           // cap on uploads during the increase phase (0 = unlimited, bounded by IncreaseTimeout)
	MaxCommittedDepth uint8         // committedDepth to reach before stopping uploads (mirrors the load check's stop condition)
	WarmupWait        time.Duration // max wait for nodes to leave warmup before staging
	IncreaseTimeout   time.Duration // max time to reach MaxCommittedDepth
	SettleWait        time.Duration // wait after uploads for pushsync overshoot to drain
	DecreaseTimeout   time.Duration // max time to observe a decrease after uploads stop
	RecoveryWait      time.Duration // observe mode: max wait for pull-sync recovery after each decrease
	PollInterval      time.Duration
	StakeAmount       string        // per-node stake to ensure before driving (wei, e.g. "100000000000000000"); empty/"0" = skip
	StakeGroups       []string      // node groups to stake (empty = all full nodes)
	StakeConfirmWait  time.Duration // max wait for a deposit to reflect on-chain
}

// NewDefaultOptions returns new default options.
func NewDefaultOptions() Options {
	return Options{
		Mode:              ModeDrive,
		Duration:          12 * time.Hour,
		RndSeed:           time.Now().UnixNano(),
		PostageTTL:        24 * time.Hour,
		PostageDepth:      22,
		PostageLabel:      "reserve-radius",
		UploadGroups:      []string{"bee"},
		BlobSize:          1 << 20, // 1 MiB
		MaxUploads:        60,
		MaxCommittedDepth: 2,
		WarmupWait:        15 * time.Minute,
		IncreaseTimeout:   5 * time.Minute,
		SettleWait:        time.Minute,
		DecreaseTimeout:   20 * time.Minute,
		RecoveryWait:      5 * time.Minute,
		PollInterval:      15 * time.Second,
		StakeAmount:       "", // skip staking unless set
		StakeGroups:       nil,
		StakeConfirmWait:  2 * time.Minute,
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

// Mode values for Options.Mode.
const (
	ModeDrive   = "drive"   // upload to force a radius change, then observe the decrease
	ModeObserve = "observe" // monitor radius changes driven externally (e.g. by the load check)
)

// Run dispatches on Mode: drive (force a change and observe it) or observe
// (monitor changes driven externally, e.g. by a parallel load check).
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return errors.New("invalid options type")
	}
	switch o.Mode {
	case "", ModeDrive:
		return c.runDrive(ctx, cluster, o)
	case ModeObserve:
		return c.runObserve(ctx, cluster, o)
	default:
		return fmt.Errorf("invalid mode %q (want %q or %q)", o.Mode, ModeDrive, ModeObserve)
	}
}

// selectNodes returns the observed full-node clients (shuffled, optionally filtered to UploadGroups).
func (c *Check) selectNodes(ctx context.Context, cluster orchestration.Cluster, o Options) (orchestration.ClientList, error) {
	c.logger.Infof("random seed: %d", o.RndSeed)
	rnd := random.PseudoGenerator(o.RndSeed)
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

// ensureStaked makes sure every node in the staking set has at least StakeAmount staked,
// depositing the shortfall and confirming on-chain. It is idempotent: a node already
// at/above the target is skipped. It is a no-op when StakeAmount is empty or "0". The
// staking set is the full nodes in StakeGroups, or — when StakeGroups is empty — all full nodes.
func (c *Check) ensureStaked(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	amount := strings.TrimSpace(o.StakeAmount)
	if amount == "" || amount == "0" {
		return nil // staking disabled
	}
	want, ok := new(big.Int).SetString(amount, 10)
	if !ok || want.Sign() <= 0 {
		return fmt.Errorf("invalid stake-amount %q (want a positive base-10 wei integer)", o.StakeAmount)
	}

	// Staking touches every node in the set regardless of order, so no shuffle is needed.
	full, err := cluster.FullNodeClients(ctx)
	if err != nil {
		return fmt.Errorf("get full node clients for staking: %w", err)
	}
	// Empty StakeGroups → stake all full nodes; otherwise just the named groups.
	nodes := full.FilterByNodeGroups(o.StakeGroups)

	c.logger.Infof("ensureStaked: ensuring >= %s wei staked on %d node(s)", want, len(nodes))
	for _, n := range nodes {
		if err := c.ensureNodeStaked(ctx, n, want, o); err != nil {
			return fmt.Errorf("ensureStaked %s: %w", n.Name(), err)
		}
	}
	return nil
}

// ensureNodeStaked tops a single node up to want (if currently below) and confirms the deposit.
func (c *Check) ensureNodeStaked(ctx context.Context, n *bee.Client, want *big.Int, o Options) error {
	cur, err := n.GetStake(ctx)
	if err != nil {
		return fmt.Errorf("get stake: %w", err)
	}
	if cur.Cmp(want) >= 0 {
		c.logger.Infof("ensureStaked: %s already staked %s wei (>= %s) — skip", n.Name(), cur, want)
		return nil
	}
	c.logger.Infof("ensureStaked: %s staked %s wei < target %s — depositing", n.Name(), cur, want)
	if _, err := n.DepositStake(ctx, want); err != nil {
		return fmt.Errorf("deposit stake: %w", err)
	}
	// POST /stake returns a tx hash; the staked amount may not reflect until the tx
	// mines (~a few blocks), so poll until it confirms.
	return c.waitStakeAtLeast(ctx, n, want, o)
}

// waitStakeAtLeast polls a node's stake until it reaches want or StakeConfirmWait expires.
func (c *Check) waitStakeAtLeast(ctx context.Context, n *bee.Client, want *big.Int, o Options) error {
	deadline := time.Now().Add(o.StakeConfirmWait)
	var last error
	for {
		cur, err := n.GetStake(ctx)
		last = err
		if err == nil && cur.Cmp(want) >= 0 {
			c.logger.Infof("ensureStaked: %s confirmed staked %s wei", n.Name(), cur)
			return nil
		}
		if time.Now().After(deadline) {
			if last != nil {
				return fmt.Errorf("stake not confirmed within %s: %w", o.StakeConfirmWait, last)
			}
			return fmt.Errorf("stake not confirmed within %s (still below %s wei)", o.StakeConfirmWait, want)
		}
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err
		}
	}
}

// runDrive uploads to force a radius increase, then observes the decrease + recovery.
func (c *Check) runDrive(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	if o.MaxCommittedDepth == 0 {
		return errors.New("max-committed-depth must be > 0")
	}
	nodes, err := c.selectNodes(ctx, cluster, o)
	if err != nil {
		return err
	}
	uploader := nodes[0] // a random node, since the list is shuffled
	c.logger.Infof("mode=drive uploader: %s, observing %d node(s)", uploader.Name(), len(nodes))

	// 0. Ensure the configured node groups are staked (no-op unless StakeAmount is set).
	if err := c.ensureStaked(ctx, cluster, o); err != nil {
		return err
	}

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

// obsNode is the per-node state the observe monitor tracks across polls.
type obsNode struct {
	lastRadius uint8
	haveRadius bool
	recoverBy  time.Time // non-zero => awaiting recovery after a decrease
	downAt     time.Time // when the pending decrease started (for time-to-recovery)
	frozen     bool      // currently inside a frozen episode
}

// runObserve monitors a cluster for Duration without uploading; the radius is driven
// externally (e.g. by a parallel load check). It records every up/down radius
// transition and, after each decrease, asserts the node recovers — preferring the
// redistribution-game signal (isFullySynced, not frozen) over the weaker pullsyncRate
// when /redistributionstate is available. It also flags freeze episodes directly
// (a frozen node skips rounds — the October halt symptom). Halt indicators
// (un-recovered decreases or freezes) fail the check at the end.
func (c *Check) runObserve(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	nodes, err := c.selectNodes(ctx, cluster, o)
	if err != nil {
		return err
	}
	c.logger.Infof("mode=observe monitoring %d node(s) for %s (no uploads; drive the radius externally, e.g. the load check)", len(nodes), o.Duration)

	// Ensure the configured node groups are staked (no-op unless StakeAmount is set).
	if err := c.ensureStaked(ctx, cluster, o); err != nil {
		return err
	}

	if err := c.waitForWarmupDone(ctx, nodes, o); err != nil {
		return err
	}

	st := make(map[string]*obsNode, len(nodes))
	for _, n := range nodes {
		st[n.Name()] = &obsNode{}
	}
	redistAvailable := true // flips false on first error; selects which recovery signal to use
	unrecovered, freezes := 0, 0
	c.logger.Info("observe: watching radius transitions, freezes, and redistribution liveness")

	deadline := time.Now().Add(o.Duration)
	for time.Now().Before(deadline) {
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err // parent context cancelled (e.g. check timeout)
		}
		for _, n := range nodes {
			name := n.Name()
			ns := st[name]

			s, err := n.Status(ctx)
			if err != nil {
				continue
			}
			c.emit(name, s)

			// redistribution-game state (full-mode only; best-effort)
			var rs *api.RedistributionState
			if redistAvailable {
				if r, rerr := n.RedistributionState(ctx); rerr == nil {
					rs = r
					c.emitRedist(name, r)
				} else {
					redistAvailable = false
					c.logger.Warningf("observe: /redistributionstate unavailable (%v); using pullsyncRate for recovery, no freeze detection", rerr)
				}
			}

			// radius transition
			cur := s.StorageRadius
			if ns.haveRadius && cur != ns.lastRadius {
				dir := "up"
				if cur < ns.lastRadius {
					dir = "down"
				}
				c.metrics.RadiusTransitions.WithLabelValues(name, dir).Inc()
				c.logger.Infof("observe: %s storageRadius %d -> %d (%s) within=%d pullsyncRate=%.4f", name, ns.lastRadius, cur, dir, s.ReserveSizeWithinRadius, s.PullsyncRate)
				if dir == "down" {
					ns.recoverBy = time.Now().Add(o.RecoveryWait)
					ns.downAt = time.Now()
				}
			}
			ns.lastRadius = cur
			ns.haveRadius = true

			// freeze detection — a frozen node skips redistribution rounds (halt symptom)
			if rs != nil {
				switch {
				case rs.IsFrozen && !ns.frozen:
					ns.frozen = true
					freezes++
					c.logger.Warningf("observe: %s is FROZEN at round %d (halt symptom; lastFrozenRound=%d)", name, rs.Round, rs.LastFrozenRound)
				case !rs.IsFrozen:
					ns.frozen = false
				}
			}

			// recovery tracking after a decrease
			if !ns.recoverBy.IsZero() {
				recovered, reason := false, ""
				switch {
				case rs != nil && rs.IsFullySynced && !rs.IsFrozen:
					recovered, reason = true, "fullySynced"
				case rs == nil && s.PullsyncRate > 0:
					recovered, reason = true, "pullsyncRate(fallback)"
				}
				switch {
				case recovered:
					c.metrics.RecoveryObserved.WithLabelValues(name, "recovered").Inc()
					c.metrics.TimeToFullySynced.Set(time.Since(ns.downAt).Seconds())
					c.logger.Infof("observe: %s recovered after decrease in %s (%s)", name, time.Since(ns.downAt).Round(time.Second), reason)
					ns.recoverBy = time.Time{}
				case time.Now().After(ns.recoverBy):
					unrecovered++
					c.metrics.RecoveryObserved.WithLabelValues(name, "timeout").Inc()
					c.logger.Warningf("observe: %s did NOT recover within %s after decrease (not fullySynced / no pull-sync) — pull-sync halt symptom (cf. PR #581)", name, o.RecoveryWait)
					ns.recoverBy = time.Time{}
				}
			}
		}
	}

	if unrecovered > 0 || freezes > 0 {
		return fmt.Errorf("observe: halt indicators over %s — %d decrease(s) without recovery, %d freeze episode(s)", o.Duration, unrecovered, freezes)
	}
	c.logger.Infof("observe: completed %s monitor — no halt indicators (all decreases recovered, no freezes)", o.Duration)
	return nil
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

// driveIncrease uploads blobs to the uploader until ALL observed nodes report
// committedDepth >= MaxCommittedDepth, confirmed over consecutive polls. Gating on the
// cluster minimum (not just the uploader, which reaches the target first) gives pull-sync
// time to bring slow peers up. It uses the same committedDepth signal as the load check
// (committedDepth drives the storage radius up deterministically, whereas reading
// storageRadius directly lags), and tracks the per-node peak storageRadius along the way
// so observeDecrease can later detect the radius falling back.
func (c *Check) driveIncrease(ctx context.Context, uploader *bee.Client, nodes orchestration.ClientList, batchID string, o Options, peak map[string]uint8) error {
	// requiredConfirmations: the cluster must report min committedDepth >= target on this
	// many consecutive polls before we stop — debounces transients and gives pull-sync time
	// to bring slow peers up to the target (the uploader reaches it first).
	const requiredConfirmations = 3

	maxDesc := "unlimited"
	if o.MaxUploads > 0 {
		maxDesc = fmt.Sprintf("%d", o.MaxUploads)
	}
	c.logger.Infof("driving increase: %d-byte blobs to %s until ALL nodes committedDepth>=%d for %d consecutive checks (max %s uploads, timeout %s)",
		o.BlobSize, uploader.Name(), o.MaxCommittedDepth, requiredConfirmations, maxDesc, o.IncreaseTimeout)
	start := time.Now()
	deadline := start.Add(o.IncreaseTimeout)
	t := test.NewTest(c.logger)
	data := make([]byte, o.BlobSize)

	uploads := 0
	confirmations := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("not all nodes reached committedDepth %d within %s (%d uploads) — is the bee reserve patch active and is there enough data?", o.MaxCommittedDepth, o.IncreaseTimeout, uploads)
		}

		mx, minCD, allReachable := c.updatePeak(ctx, nodes, peak)

		// Stop once every node has reached the target, confirmed over consecutive polls.
		if allReachable && minCD >= o.MaxCommittedDepth {
			confirmations++
			c.logger.Infof("all nodes committedDepth>=%d (confirmation %d/%d, maxStorageRadius=%d)", o.MaxCommittedDepth, confirmations, requiredConfirmations, mx)
			if confirmations >= requiredConfirmations {
				c.metrics.TimeToIncrease.Set(time.Since(start).Seconds())
				c.logger.Infof("cluster reached committedDepth %d (all nodes, %dx confirmed) after %d uploads (~%.1f MiB) in %s",
					o.MaxCommittedDepth, requiredConfirmations, uploads, float64(int64(uploads)*o.BlobSize)/(1<<20), time.Since(start).Round(time.Second))
				return nil
			}
			// Don't upload during confirmation; space the checks so pull-sync keeps peers caught up.
			if err := sleepCtx(ctx, o.PollInterval); err != nil {
				return err
			}
			continue
		}

		// A node is still below target → reset the streak and push more data.
		if confirmations > 0 {
			c.logger.Infof("min committedDepth fell to %d (<%d) — resetting confirmation streak", minCD, o.MaxCommittedDepth)
		}
		confirmations = 0
		if o.MaxUploads > 0 && uploads >= o.MaxUploads {
			return fmt.Errorf("not all nodes reached committedDepth %d after %d uploads (min=%d)", o.MaxCommittedDepth, uploads, minCD)
		}
		if _, err := crand.Read(data); err != nil {
			return fmt.Errorf("generate random data: %w", err)
		}
		if _, _, err := t.Upload(ctx, uploader, data, batchID, nil); err != nil {
			c.logger.Errorf("upload #%d failed: %v", uploads+1, err)
			continue
		}
		uploads++
		c.logger.Infof("increase: upload #%d, minCommittedDepth=%d (target %d), maxStorageRadius=%d", uploads, minCD, o.MaxCommittedDepth, mx)
	}
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

// updatePeak polls every node, emits metrics, and raises the per-node high-water peak
// storageRadius. It reports the raw observations — the max storageRadius seen, the minimum
// committedDepth across the reachable nodes (0 if none responded), and whether every node
// responded — and leaves the "whole cluster reached target" decision to the caller, which
// combines minCommittedDepth with allReachable.
func (c *Check) updatePeak(ctx context.Context, nodes orchestration.ClientList, peak map[string]uint8) (maxRadius, minCommittedDepth uint8, allReachable bool) {
	allReachable = true
	minCD := uint8(math.MaxUint8)
	for _, n := range nodes {
		s, err := n.Status(ctx)
		if err != nil {
			c.logger.Debugf("%s: status error: %v", n.Name(), err)
			allReachable = false
			continue
		}
		c.emit(n.Name(), s)
		if s.StorageRadius > peak[n.Name()] {
			peak[n.Name()] = s.StorageRadius
		}
		if s.StorageRadius > maxRadius {
			maxRadius = s.StorageRadius
		}
		if s.CommittedDepth < minCD {
			minCD = s.CommittedDepth
		}
		c.logger.Infof("%s: storageRadius=%d (peak %d) committedDepth=%d reserveSize=%d withinR=%d pullsyncRate=%.4f",
			n.Name(), s.StorageRadius, peak[n.Name()], s.CommittedDepth, s.ReserveSize, s.ReserveSizeWithinRadius, s.PullsyncRate)
	}
	if minCD == math.MaxUint8 {
		minCD = 0 // no node responded
	}
	return maxRadius, minCD, allReachable
}

func (c *Check) emit(node string, s *api.StatusResponse) {
	c.metrics.StorageRadius.WithLabelValues(node).Set(float64(s.StorageRadius))
	c.metrics.ReserveSize.WithLabelValues(node).Set(float64(s.ReserveSize))
	c.metrics.ReserveWithinRadius.WithLabelValues(node).Set(float64(s.ReserveSizeWithinRadius))
	c.metrics.PullsyncRate.WithLabelValues(node).Set(s.PullsyncRate)
}

func (c *Check) emitRedist(node string, r *api.RedistributionState) {
	c.metrics.FullySynced.WithLabelValues(node).Set(b2f(r.IsFullySynced))
	c.metrics.Frozen.WithLabelValues(node).Set(b2f(r.IsFrozen))
	c.metrics.RedistRound.WithLabelValues(node).Set(float64(r.Round))
	c.metrics.LastSampleDuration.WithLabelValues(node).Set(r.LastSampleDurationSeconds)
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

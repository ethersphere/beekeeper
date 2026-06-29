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
	"math/big"
	"strconv"
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
	Mode             string        // "drive" (upload to force a change) or "observe" (monitor a change driven externally)
	Duration         time.Duration // observe mode: total monitor run length
	RndSeed          int64
	StakeAmount      string        // per-node stake to ensure before driving (wei, e.g. "100000000000000000"); empty/"0" = skip
	StakeGroups      []string      // node groups to stake (empty = the observed/selected groups)
	StakeConfirmWait time.Duration // max wait for a deposit to reflect on-chain (~10 blocks)
	PostageTTL       time.Duration
	PostageDepth     uint64
	PostageLabel     string
	UploadGroups     []string      // node groups to upload to / observe (empty = all full nodes)
	BlobSize         int64         // bytes per upload
	MaxUploads       int           // cap on uploads during the increase phase
	TargetRadius     uint8         // drive mode: storageRadius any node must reach before stopping uploads
	DisruptAtRadius  uint8         // halt mode: storageRadius ALL observed nodes must reach before disruption
	WarmupWait       time.Duration // max wait for nodes to leave warmup before staging
	IncreaseTimeout  time.Duration // max time to reach TargetRadius
	SettleWait       time.Duration // wait after uploads for pushsync overshoot to drain
	DecreaseTimeout  time.Duration // max time to observe a decrease after uploads stop
	RecoveryWait     time.Duration // observe mode: max wait for pull-sync recovery after each decrease
	PollInterval     time.Duration
}

// NewDefaultOptions returns new default options.
func NewDefaultOptions() Options {
	return Options{
		Mode:             ModeDrive,
		Duration:         12 * time.Hour,
		RndSeed:          time.Now().UnixNano(),
		StakeAmount:      "", // skip staking unless set
		StakeGroups:      nil,
		StakeConfirmWait: 2 * time.Minute,
		PostageTTL:       24 * time.Hour,
		PostageDepth:     22,
		PostageLabel:     "reserve-radius",
		UploadGroups:     []string{"bee"},
		BlobSize:         1 << 20, // 1 MiB
		MaxUploads:       60,
		TargetRadius:     1,
		DisruptAtRadius:  3,
		WarmupWait:       15 * time.Minute,
		IncreaseTimeout:  5 * time.Minute,
		SettleWait:       time.Minute,
		DecreaseTimeout:  20 * time.Minute,
		RecoveryWait:     5 * time.Minute,
		PollInterval:     15 * time.Second,
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
	ModeHalt    = "halt"    // self-driving: stake → drive all nodes → disrupt → observe outcome
)

// Run dispatches on Mode: drive (force a change and observe it), observe
// (monitor changes driven externally, e.g. by a parallel load check), or halt
// (self-driving stake → drive → disrupt → observe-outcome reproduction).
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
	case ModeHalt:
		return c.runHalt(ctx, cluster, o)
	default:
		return fmt.Errorf("invalid mode %q (want %q, %q or %q)", o.Mode, ModeDrive, ModeObserve, ModeHalt)
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

// ensureStaked makes sure every node in the staking set has at least StakeAmount
// staked, depositing the shortfall and confirming on-chain. It is idempotent: a
// node already at/above the target is skipped, and "already staked" is tolerated.
// Skips entirely when StakeAmount is empty or "0". The staking set is StakeGroups
// (full nodes filtered) or, when unset, the observed nodes.
func (c *Check) ensureStaked(ctx context.Context, cluster orchestration.Cluster, observed orchestration.ClientList, o Options) error {
	amount := strings.TrimSpace(o.StakeAmount)
	if amount == "" || amount == "0" {
		return nil // staking disabled
	}
	want, ok := new(big.Int).SetString(amount, 10)
	if !ok || want.Sign() <= 0 {
		return fmt.Errorf("invalid stake-amount %q (want a positive base-10 wei integer)", o.StakeAmount)
	}

	nodes := observed
	if len(o.StakeGroups) > 0 {
		rnd := random.PseudoGenerator(o.RndSeed)
		full, err := cluster.ShuffledFullNodeClients(ctx, rnd)
		if err != nil {
			return fmt.Errorf("get full node clients for staking: %w", err)
		}
		nodes = full.FilterByNodeGroups(o.StakeGroups)
	}
	if len(nodes) == 0 {
		return errors.New("ensureStaked: no nodes selected for staking")
	}

	c.logger.Infof("ensureStaked: ensuring >= %s wei staked on %d node(s)", want, len(nodes))
	for _, n := range nodes {
		if err := c.ensureNodeStaked(ctx, n, want, o); err != nil {
			return fmt.Errorf("ensureStaked %s: %w", n.Name(), err)
		}
	}
	return nil
}

// ensureNodeStaked tops a single node up to want (if below) and confirms the deposit.
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
	// mines (~a few blocks), so poll until it confirms (or already mined → immediate).
	return c.waitStakeAtLeast(ctx, n, want, o)
}

// waitStakeAtLeast polls a node's stake until it reaches want or the budget expires.
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
	if o.TargetRadius == 0 {
		return errors.New("target-radius must be > 0")
	}
	nodes, err := c.selectNodes(ctx, cluster, o)
	if err != nil {
		return err
	}
	uploader := nodes[0] // a random node, since the list is shuffled
	c.logger.Infof("mode=drive uploader: %s, observing %d node(s)", uploader.Name(), len(nodes))

	// 0. Stake (optional) — runs before warmup so deposits confirm while we wait.
	if err := c.ensureStaked(ctx, cluster, nodes, o); err != nil {
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

// runHalt self-drives the whole reproduction: stake (optional) → warmup → drive
// ALL observed nodes to DisruptAtRadius → settle → disrupt → observe the outcome.
// The disruption (Phase 3) and outcome classification/verdict (Phase 4) stages
// land in later cycles; for now it completes the stake/drive/settle prefix.
func (c *Check) runHalt(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	if o.DisruptAtRadius == 0 {
		return errors.New("disrupt-at-radius must be > 0")
	}
	nodes, err := c.selectNodes(ctx, cluster, o)
	if err != nil {
		return err
	}
	uploader := nodes[0] // a random node, since the list is shuffled
	c.logger.Infof("mode=halt uploader: %s, %d observed node(s), disrupt-at-radius=%d", uploader.Name(), len(nodes), o.DisruptAtRadius)

	// 0. Stake (optional) — runs before warmup so deposits confirm while we wait.
	if err := c.ensureStaked(ctx, cluster, nodes, o); err != nil {
		return err
	}

	// 1. Wait for warmup/stabilization.
	if err := c.waitForWarmupDone(ctx, nodes, o); err != nil {
		return err
	}
	c.baselineSnapshot(ctx, nodes)

	// 2. Drive every observed node up to DisruptAtRadius (a populated neighbourhood
	//    is the precondition for the disruption to bite).
	batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("create batch on %s: %w", uploader.Name(), err)
	}
	c.logger.WithField("batch_id", batchID).Infof("node %s: using batch", uploader.Name())
	peak := make(map[string]uint8, len(nodes))
	if err := c.driveAllToRadius(ctx, uploader, nodes, batchID, o, peak); err != nil {
		return err
	}

	// 3. Settle window — let pushsync overshoot drain before disrupting.
	c.logger.Infof("halt: all nodes at radius %d; settling for %s before disruption", o.DisruptAtRadius, o.SettleWait)
	settleDeadline := time.Now().Add(o.SettleWait)
	for time.Now().Before(settleDeadline) {
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err
		}
		c.updatePeak(ctx, nodes, peak)
	}

	// TODO(Phase 3): disrupt the neighbourhood (node-churn / batch-expiry).
	// TODO(Phase 4): observe the outcome, classify HALT|RECOVERED|MONITORED, apply the verdict.
	c.logger.Info("halt: stake+drive+settle complete; disruption and outcome observation land in Phase 3/4")
	return nil
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

	if err := c.ensureStaked(ctx, cluster, nodes, o); err != nil {
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
		_, mx := c.updatePeak(ctx, nodes, peak)
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

// driveAllToRadius uploads blobs to the uploader until EVERY observed node reaches
// DisruptAtRadius (gates on the min, not the max — the whole neighbourhood must be
// populated before disruption). Otherwise it mirrors driveIncrease.
func (c *Check) driveAllToRadius(ctx context.Context, uploader *bee.Client, nodes orchestration.ClientList, batchID string, o Options, peak map[string]uint8) error {
	c.logger.Infof("halt drive: %d-byte blobs to %s until ALL %d node(s) storageRadius>=%d (max %d uploads, timeout %s)",
		o.BlobSize, uploader.Name(), len(nodes), o.DisruptAtRadius, o.MaxUploads, o.IncreaseTimeout)
	start := time.Now()
	deadline := start.Add(o.IncreaseTimeout)
	t := test.NewTest(c.logger)
	data := make([]byte, o.BlobSize)

	for i := 1; i <= o.MaxUploads; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("not all nodes reached storageRadius %d within %s (%d uploads) — is the bee reserve patch active and is there enough data?", o.DisruptAtRadius, o.IncreaseTimeout, i-1)
		}
		if _, err := crand.Read(data); err != nil {
			return fmt.Errorf("generate random data: %w", err)
		}
		if _, _, err := t.Upload(ctx, uploader, data, batchID, nil); err != nil {
			c.logger.Errorf("upload #%d failed: %v", i, err)
			continue
		}
		mn, mx := c.updatePeak(ctx, nodes, peak)
		c.logger.Infof("halt drive: upload #%d, min storageRadius=%d max=%d (target %d)", i, mn, mx, o.DisruptAtRadius)
		if mn >= o.DisruptAtRadius {
			c.metrics.TimeToIncrease.Set(time.Since(start).Seconds())
			c.logger.Infof("all nodes reached storageRadius %d after %d uploads (~%.1f MiB) in %s",
				o.DisruptAtRadius, i, float64(int64(i)*o.BlobSize)/(1<<20), time.Since(start).Round(time.Second))
			return nil
		}
	}
	return fmt.Errorf("not all nodes reached storageRadius %d after %d uploads", o.DisruptAtRadius, o.MaxUploads)
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

// baselineSnapshot logs and emits the pre-disruption reference state for every node:
// storage radius + reserveSizeWithinRadius (/status), isFullySynced + round
// participation (/redistributionstate), and stake. Best-effort per node — a failed
// read shows as "?" in the log. Phase 4 will extend this to return the captured
// values for onset and staked-round-loss comparison.
func (c *Check) baselineSnapshot(ctx context.Context, nodes orchestration.ClientList) {
	for _, n := range nodes {
		name := n.Name()

		radius, within := "?", "?"
		if s, err := n.Status(ctx); err == nil {
			c.emit(name, s)
			radius = strconv.Itoa(int(s.StorageRadius))
			within = strconv.FormatUint(s.ReserveSizeWithinRadius, 10)
		} else {
			c.logger.Debugf("baseline %s: status error: %v", name, err)
		}

		synced, round, played, won := "?", "?", "?", "?"
		if r, err := n.RedistributionState(ctx); err == nil {
			c.emitRedist(name, r)
			synced = strconv.FormatBool(r.IsFullySynced)
			round = strconv.FormatUint(r.Round, 10)
			played = strconv.FormatUint(r.LastPlayedRound, 10)
			won = strconv.FormatUint(r.LastWonRound, 10)
		} else {
			c.logger.Debugf("baseline %s: redistributionstate error: %v", name, err)
		}

		stake := "?"
		if st, err := n.GetStake(ctx); err == nil {
			stake = st.String()
		} else {
			c.logger.Debugf("baseline %s: stake error: %v", name, err)
		}

		c.logger.Infof("[baseline] %s: storageRadius=%s withinR=%s fullySynced=%s round=%s lastPlayed=%s lastWon=%s stake=%s",
			name, radius, within, synced, round, played, won, stake)
	}
}

// updatePeak polls each node, emits metrics, raises the per-node high-water peak,
// logs a line, and returns the min and max current storageRadius across the nodes
// that responded (min is 0 if none responded).
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
		c.logger.Infof("%s: storageRadius=%d (peak %d) reserveSize=%d withinR=%d pullsyncRate=%.4f",
			n.Name(), s.StorageRadius, peak[n.Name()], s.ReserveSize, s.ReserveSizeWithinRadius, s.PullsyncRate)
	}
	return minR, maxR
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

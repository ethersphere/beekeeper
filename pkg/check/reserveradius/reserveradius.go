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
	DisruptMechanism string        // "node-churn" (default) | "batch-expiry" | "both" | "none"
	DisruptNodeCount int           // node-churn: full nodes to remove (randomly, seeded by RndSeed); 0 = skip
	DisruptMethod    string        // node-churn: "stop" (scale-0, default) | "delete"
	MinSurvivors     int           // refuse to disrupt below this many surviving nodes
	Verdict          string        // halt mode: "report" (default, never fail on outcome) | "assert" (gate on ExpectRecovery)
	ExpectRecovery   bool          // halt mode + verdict=assert: false ⇒ expect HALT, true ⇒ expect RECOVERED
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
		DisruptMechanism: DisruptNodeChurn,
		DisruptNodeCount: 2,
		DisruptMethod:    RemoveStop,
		MinSurvivors:     3,
		Verdict:          VerdictReport,
		ExpectRecovery:   false,
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

// Disruption mechanisms for Options.DisruptMechanism.
const (
	DisruptNodeChurn   = "node-churn"   // stop/delete random full nodes (default)
	DisruptBatchExpiry = "batch-expiry" // expire a fill batch → commitment drop → radius decrease
	DisruptBoth        = "both"         // batch-expiry + node-churn for a harsher merge
	DisruptNone        = "none"         // monitor-only (no disruption)
)

// Node-removal methods for Options.DisruptMethod.
const (
	RemoveStop   = "stop"   // scale statefulset to 0 (restorable)
	RemoveDelete = "delete" // delete statefulset + services/ingress/secret/configmap
)

// Run outcomes classified by the halt observe-outcome stage.
const (
	OutcomeMonitored = "MONITORED" // no disruption requested; state recorded only
	OutcomeHalt      = "HALT"      // post-disruption sustained non-convergence and/or staked round-loss
	OutcomeRecovered = "RECOVERED" // post-disruption, all de-synced survivors re-converged
)

// Verdict policies for Options.Verdict.
const (
	VerdictReport = "report" // never fail on the outcome; only operational errors fail (default)
	VerdictAssert = "assert" // fail iff the outcome contradicts expect-recovery (A/B regression gate)
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

	// 4. Disrupt the neighbourhood; the survivor set is what the observe loop polls.
	survivors, err := c.disrupt(ctx, cluster, nodes, uploader.Name(), o)
	if err != nil {
		return err
	}
	disrupted := len(survivors) < len(nodes) // node-churn removed nodes; none/count-0 = monitor-only
	disruptedAt := time.Now()
	c.snapshot(ctx, survivors, "post-disrupt")

	// 5. Observe the outcome on the survivors and classify it.
	outcome, err := c.observeOutcome(ctx, survivors, disrupted, disruptedAt, o)
	if err != nil {
		return err
	}
	c.logger.Infof("halt: run outcome = %s", outcome)
	return c.applyVerdict(outcome, o)
}

// applyVerdict turns a classified outcome into a check result per the verdict policy.
// report (default) always succeeds on any outcome (only operational errors, raised
// earlier, fail). assert fails iff the outcome contradicts expect-recovery; MONITORED
// always passes (no disruption was requested).
func (c *Check) applyVerdict(outcome string, o Options) error {
	switch o.Verdict {
	case "", VerdictReport:
		c.logger.Infof("verdict=report: outcome %s (reported, not gated)", outcome)
		return nil
	case VerdictAssert:
		if outcome == OutcomeMonitored {
			c.logger.Info("verdict=assert: outcome MONITORED (no disruption requested) — pass")
			return nil
		}
		want := OutcomeHalt
		if o.ExpectRecovery {
			want = OutcomeRecovered
		}
		if outcome == want {
			c.logger.Infof("verdict=assert: outcome %s matches expect-recovery=%t — pass", outcome, o.ExpectRecovery)
			return nil
		}
		return fmt.Errorf("verdict=assert: outcome %s contradicts expect-recovery=%t (expected %s)", outcome, o.ExpectRecovery, want)
	default:
		return fmt.Errorf("invalid verdict %q (want %q or %q)", o.Verdict, VerdictReport, VerdictAssert)
	}
}

// disrupt applies the configured disruption mechanism and returns the survivor set
// the observe loop should poll. excludeName is kept out of node-churn selection (the
// uploader in halt mode); pass "" when there is nothing to protect.
func (c *Check) disrupt(ctx context.Context, cluster orchestration.Cluster, observed orchestration.ClientList, excludeName string, o Options) (orchestration.ClientList, error) {
	switch o.DisruptMechanism {
	case DisruptNone:
		c.logger.Info("disrupt: mechanism=none (monitor-only); no nodes removed")
		return observed, nil
	case "", DisruptNodeChurn:
		return c.disruptNodeChurn(ctx, cluster, observed, excludeName, o)
	case DisruptBatchExpiry, DisruptBoth:
		return nil, fmt.Errorf("disrupt-mechanism %q not yet implemented (Phase 3b)", o.DisruptMechanism)
	default:
		return nil, fmt.Errorf("invalid disrupt-mechanism %q (want %q, %q, %q or %q)", o.DisruptMechanism, DisruptNodeChurn, DisruptBatchExpiry, DisruptBoth, DisruptNone)
	}
}

// disruptNodeChurn randomly selects DisruptNodeCount nodes (seeded by RndSeed,
// reproducible), excluding excludeName, and removes them via Stop (scale-0) or
// Delete. The MinSurvivors guard is checked BEFORE any removal, so a too-aggressive
// count fails without touching the cluster. DisruptNodeCount <= 0 is monitor-only.
// Returns the survivor ClientList (observed minus removed).
func (c *Check) disruptNodeChurn(ctx context.Context, cluster orchestration.Cluster, observed orchestration.ClientList, excludeName string, o Options) (orchestration.ClientList, error) {
	if o.DisruptNodeCount <= 0 {
		c.logger.Info("disrupt: node-churn disrupt-node-count=0 (monitor-only); no nodes removed")
		return observed, nil
	}
	switch o.DisruptMethod {
	case "", RemoveStop, RemoveDelete:
	default:
		return nil, fmt.Errorf("invalid disrupt-method %q (want %q or %q)", o.DisruptMethod, RemoveStop, RemoveDelete)
	}

	// Candidate pool excludes the protected node (the uploader in halt mode).
	candidates := make(orchestration.ClientList, 0, len(observed))
	for _, n := range observed {
		if n.Name() != excludeName {
			candidates = append(candidates, n)
		}
	}
	if o.DisruptNodeCount > len(candidates) {
		return nil, fmt.Errorf("disrupt: cannot remove %d node(s), only %d candidate(s) available (excluding %q)", o.DisruptNodeCount, len(candidates), excludeName)
	}
	if survivors := len(observed) - o.DisruptNodeCount; survivors < o.MinSurvivors {
		return nil, fmt.Errorf("disrupt: removing %d of %d node(s) would leave %d survivor(s), below min-survivors=%d", o.DisruptNodeCount, len(observed), survivors, o.MinSurvivors)
	}

	// Reproducible random pick of DisruptNodeCount candidates.
	rnd := random.PseudoGenerator(o.RndSeed)
	pick := rnd.Perm(len(candidates))[:o.DisruptNodeCount]
	removed := make(map[string]bool, len(pick))
	nodesByName := cluster.Nodes()
	ns := cluster.Namespace()
	for _, ci := range pick {
		name := candidates[ci].Name()
		node, ok := nodesByName[name]
		if !ok {
			return nil, fmt.Errorf("disrupt: node %q not found in cluster", name)
		}
		method := o.DisruptMethod
		if method == "" {
			method = RemoveStop
		}
		var rerr error
		if method == RemoveDelete {
			rerr = node.Delete(ctx, ns)
		} else {
			rerr = node.Stop(ctx, ns)
		}
		if rerr != nil {
			return nil, fmt.Errorf("disrupt: %s node %q: %w", method, name, rerr)
		}
		removed[name] = true
		c.metrics.Disruptions.WithLabelValues(DisruptNodeChurn).Inc()
		c.logger.Warningf("disrupt: %s node %q (%d/%d) at %s", method, name, len(removed), o.DisruptNodeCount, time.Now().Format(time.RFC3339))
	}

	survivors := make(orchestration.ClientList, 0, len(observed)-len(removed))
	for _, n := range observed {
		if !removed[n.Name()] {
			survivors = append(survivors, n)
		}
	}
	c.logger.Infof("disrupt: node-churn removed %d node(s) via %s; %d survivor(s) remain", len(removed), o.DisruptMethod, len(survivors))
	return survivors, nil
}

// outcomeNode tracks a survivor's post-disruption trajectory for classification.
type outcomeNode struct {
	haveRef   bool
	refSynced bool   // was the node fullySynced at the first post-disruption reading
	refRound  uint64 // round participation reference (for staked round-loss)
	refPlayed uint64
	refWon    uint64

	onset      bool      // lost sync (fullySynced true->false) after disruption
	onsetAt    time.Time // when the onset was first seen
	recovered  bool      // returned to fullySynced WITHIN RecoveryWait after an onset
	lateResync bool      // re-synced, but only after RecoveryWait (too slow — a halt symptom)
	recoverBy  time.Time // recovery deadline after onset (onsetAt + RecoveryWait)

	haveLast   bool
	lastRound  uint64
	lastPlayed uint64
	lastWon    uint64
}

// observeOutcome polls the survivor set for Duration after disruption, tracking per
// node the onset of de-sync (fullySynced true->false), recovery (back to fullySynced
// within RecoveryWait), and staked round-loss (Round advancing while
// LastPlayedRound/LastWonRound stall). It classifies the run as MONITORED (no
// disruption), HALT (a survivor stuck de-synced past RecoveryWait and/or round-loss),
// or RECOVERED (all de-synced survivors re-converged), emits the outcome metric + a
// summary, and returns the outcome. The verdict policy is applied by the caller.
func (c *Check) observeOutcome(ctx context.Context, survivors orchestration.ClientList, disrupted bool, disruptedAt time.Time, o Options) (string, error) {
	st := make(map[string]*outcomeNode, len(survivors))
	for _, n := range survivors {
		st[n.Name()] = &outcomeNode{}
	}
	c.logger.Infof("observe-outcome: watching %d survivor(s) for %s (disrupted=%t, recovery-wait=%s)", len(survivors), o.Duration, disrupted, o.RecoveryWait)

	deadline := time.Now().Add(o.Duration)
	for time.Now().Before(deadline) {
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return "", err // parent context cancelled (e.g. check timeout)
		}
		for _, n := range survivors {
			name := n.Name()
			ns := st[name]

			s, err := n.Status(ctx)
			if err != nil {
				continue
			}
			c.emit(name, s)

			r, rerr := n.RedistributionState(ctx)
			if rerr != nil {
				continue // the redistribution signal is the classifier; skip this poll for the node
			}
			c.emitRedist(name, r)

			if !ns.haveRef {
				ns.haveRef = true
				ns.refSynced = r.IsFullySynced
				ns.refRound, ns.refPlayed, ns.refWon = r.Round, r.LastPlayedRound, r.LastWonRound
			}
			ns.haveLast = true
			ns.lastRound, ns.lastPlayed, ns.lastWon = r.Round, r.LastPlayedRound, r.LastWonRound

			// onset: a node that was synced loses sync after disruption
			if disrupted && ns.refSynced && !ns.onset && !r.IsFullySynced {
				ns.onset = true
				ns.onsetAt = time.Now()
				ns.recoverBy = ns.onsetAt.Add(o.RecoveryWait)
				c.metrics.OnsetSeconds.WithLabelValues(name).Set(ns.onsetAt.Sub(disruptedAt).Seconds())
				c.logger.Warningf("observe-outcome: %s de-synced (fullySynced=false) at round %d, %s after disruption — onset", name, r.Round, ns.onsetAt.Sub(disruptedAt).Round(time.Second))
			}
			// recovery: back to fullySynced (and not frozen) WITHIN RecoveryWait. A
			// re-sync after the deadline is too slow to count — it stays not-recovered
			// so it classifies as stuck (the "puller gave up at SyncRate=0" false-sync
			// is exactly this: it re-flags fullySynced late without truly converging).
			if ns.onset && !ns.recovered && !ns.lateResync && r.IsFullySynced && !r.IsFrozen {
				if time.Now().Before(ns.recoverBy) {
					ns.recovered = true
					c.logger.Infof("observe-outcome: %s re-converged %s after onset (within recovery-wait)", name, time.Since(ns.onsetAt).Round(time.Second))
				} else {
					ns.lateResync = true
					c.logger.Warningf("observe-outcome: %s re-synced %s after onset — PAST recovery-wait %s, too slow (counts as halt)", name, time.Since(ns.onsetAt).Round(time.Second), o.RecoveryWait)
				}
			}
		}
	}

	return c.classifyOutcome(st, disrupted, o), nil
}

// classifyOutcome reduces the per-node trajectories to a single run outcome and emits it.
func (c *Check) classifyOutcome(st map[string]*outcomeNode, disrupted bool, o Options) string {
	if !disrupted {
		c.setOutcome(OutcomeMonitored)
		c.logger.Infof("observe-outcome: MONITORED — no disruption requested; %d node(s) observed for %s", len(st), o.Duration)
		return OutcomeMonitored
	}

	desynced, recovered, stuck, roundLoss := 0, 0, 0, 0
	for name, ns := range st {
		if ns.onset {
			desynced++
		}
		if ns.onset && ns.recovered {
			recovered++
		}
		// stuck: de-synced, never recovered, past its recovery deadline
		if ns.onset && !ns.recovered && !ns.recoverBy.IsZero() && time.Now().After(ns.recoverBy) {
			stuck++
		}
		// staked round-loss: rounds advanced after onset but the node never played/won
		if ns.onset && ns.haveLast && ns.lastRound > ns.refRound && ns.lastPlayed == ns.refPlayed && ns.lastWon == ns.refWon {
			roundLoss++
			c.metrics.RoundLoss.WithLabelValues(name).Inc()
			c.logger.Warningf("observe-outcome: %s round-loss — round %d->%d while lastPlayed/Won stalled (%d/%d) since de-sync", name, ns.refRound, ns.lastRound, ns.refPlayed, ns.refWon)
		}
	}

	if stuck > 0 || roundLoss > 0 {
		c.setOutcome(OutcomeHalt)
		c.logger.Warningf("observe-outcome: HALT — %d/%d survivor(s) stuck de-synced past %s, %d with staked round-loss (%d de-synced, %d recovered)", stuck, len(st), o.RecoveryWait, roundLoss, desynced, recovered)
		return OutcomeHalt
	}
	c.setOutcome(OutcomeRecovered)
	c.logger.Infof("observe-outcome: RECOVERED — all %d survivor(s) converged (%d de-synced then recovered)", len(st), recovered)
	return OutcomeRecovered
}

// setOutcome one-hot-encodes the classified outcome into the outcome gauge.
func (c *Check) setOutcome(outcome string) {
	for _, name := range []string{OutcomeMonitored, OutcomeHalt, OutcomeRecovered} {
		v := 0.0
		if name == outcome {
			v = 1
		}
		c.metrics.Outcome.WithLabelValues(name).Set(v)
	}
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
// disruptionActive reports whether the configured mechanism will actually disrupt, so
// observe mode runs the staged reproduction rather than the plain soak monitor.
// Monitor-only = disrupt-mechanism none, or node-churn with disrupt-node-count 0.
func disruptionActive(o Options) bool {
	switch o.DisruptMechanism {
	case DisruptNone:
		return false
	case "", DisruptNodeChurn:
		return o.DisruptNodeCount > 0
	default: // batch-expiry / both
		return true
	}
}

// runObserveDisrupt is the parallel-with-load shape: the radius is driven externally
// (e.g. by a parallel load check), so this check waits for the neighbourhood to
// populate, snapshots a baseline, disrupts, then observes + classifies the outcome —
// the same stages as halt mode, minus the self-driving upload.
func (c *Check) runObserveDisrupt(ctx context.Context, cluster orchestration.Cluster, nodes orchestration.ClientList, o Options) error {
	if o.DisruptAtRadius == 0 {
		return errors.New("disrupt-at-radius must be > 0")
	}
	c.logger.Infof("mode=observe+disrupt: staged reproduction (mechanism=%s, count=%d); radius driven externally", o.DisruptMechanism, o.DisruptNodeCount)

	if err := c.waitAllReachRadius(ctx, nodes, o); err != nil {
		return err
	}
	c.baselineSnapshot(ctx, nodes)

	survivors, err := c.disrupt(ctx, cluster, nodes, "", o) // no uploader to protect in observe mode
	if err != nil {
		return err
	}
	disrupted := len(survivors) < len(nodes)
	disruptedAt := time.Now()
	c.snapshot(ctx, survivors, "post-disrupt")

	outcome, err := c.observeOutcome(ctx, survivors, disrupted, disruptedAt, o)
	if err != nil {
		return err
	}
	c.logger.Infof("observe+disrupt: run outcome = %s", outcome)
	return c.applyVerdict(outcome, o)
}

// waitAllReachRadius blocks until every node reports storageRadius >= DisruptAtRadius
// (driven externally, e.g. by a parallel load check) or IncreaseTimeout elapses.
func (c *Check) waitAllReachRadius(ctx context.Context, nodes orchestration.ClientList, o Options) error {
	c.logger.Infof("observe+disrupt: waiting up to %s for all %d node(s) to reach storageRadius>=%d (driven externally)", o.IncreaseTimeout, len(nodes), o.DisruptAtRadius)
	deadline := time.Now().Add(o.IncreaseTimeout)
	peak := make(map[string]uint8, len(nodes))
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		mn, mx := c.updatePeak(ctx, nodes, peak)
		if mn >= o.DisruptAtRadius {
			c.logger.Infof("observe+disrupt: all nodes reached storageRadius %d (max %d)", o.DisruptAtRadius, mx)
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("not all nodes reached storageRadius %d within %s (min=%d) — is the radius being driven (e.g. the load check) and is the reserve patch active?", o.DisruptAtRadius, o.IncreaseTimeout, mn)
		}
		if err := sleepCtx(ctx, o.PollInterval); err != nil {
			return err
		}
	}
}

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

	// observe + disruption configured → run the staged reproduction (parallel-with-load
	// shape) instead of the plain soak monitor. Monitor-only requires disrupt-mechanism:
	// none (or node-churn with disrupt-node-count: 0).
	if disruptionActive(o) {
		return c.runObserveDisrupt(ctx, cluster, nodes, o)
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

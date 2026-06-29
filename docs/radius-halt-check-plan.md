# Reserve-radius check — halt-reproduction mode (plan)

Status: **Plan only — not started.** Branch `ljubisa-radius-soak`.

Goal: fold the now-working **manual** pull-sync-halt reproduction (see `docs/radius-halt.md`)
into the `reserveradius` check so the whole flow runs **unattended in one longer run** —
stake → drive radius to N → disrupt the neighbourhood (**node churn and/or batch expiry**) → observe
convergence/halt + the staked redistribution round-loss — and so the same check can serve the
**A/B harness** (stock master vs
`pullsync-optimal-design`). Optionally run alongside the `load` check in parallel.

This plan only covers the **check code** (`pkg/check/reserveradius`). Environment setup —
building the patched bee image, deploying + funding the cluster — stays manual / in the
`cluster-up` flow and is out of scope here.

## What the manual flow does today (the target to automate)

From `docs/radius-halt.md` "Staked run", the reproduced recipe is:

1. Stake every full node (`POST /stake/1e17`, verify `stakedAmount`).
2. Drive the radius to **3** with `ci-load-soak` (`max-committed-depth: 3`); stop the load once
   all nodes report `storageRadius=3`.
3. **Remove 2 of 6 full nodes** (`kubectl delete statefulset/pod`) to disrupt the populated
   neighbourhood.
4. Observe the 4 survivors ≥25 min: delayed onset (~8 min) → `within_radius` 0→~2200,
   `fullySynced→false`, `pullsyncRate` decays toward 0, split-brain end-state, never re-converges.
5. Confirm the staked round-loss: `is_playing_errors` cluster-wide; un-synced nodes can't produce a
   `ReserveSample`, so they win nothing while rounds keep advancing.

The check already does step 2's driving (`mode: drive`) and a generic observe loop (`mode: observe`),
but has **no** staking, **no** node removal, and asserts only a *radius decrease + recovery* — not the
disruption-driven non-convergence we actually reproduce.

## Capabilities confirmed (no blockers)

- **Node removal — supported.** `orchestration.Node.Delete(ctx, namespace)` (deletes statefulset +
  services/ingress/secret/configmap) and `Node.Stop(ctx, namespace)` (scales statefulset to 0); also
  `NodeGroup.DeleteNode(ctx, name)`. The check reaches them via `cluster.Nodes()` /
  `cluster.NodeGroup(name)`, and the namespace via `cluster.Namespace()`.
- **Staking — supported.** `bee.Client.DepositStake(ctx, *big.Int)` → `POST /stake/{amount}`,
  `GetStake(ctx)` → `GET /stake`. The `stake` check already uses these via `cluster.NodesClients`.
- **Batch expiry — supported (the original October lever).** `CreatePostageBatch(ctx, amount, depth,
  label)` (explicit amount → controllable expiry; batches are mutable) and `GetOrCreateMutableBatch`
  (TTL→amount via chainstate price). Read expiry via `PostageStamp(batchID).BatchTTL`/`Exists` and
  `GetChainState().TotalAmount`. Bee side (confirmed): expiry → `EvictBatch` → `evictExpiredBatches`
  → commitment drop → gated radius decrease in `pkg/storer/reserve.go`. No in-repo `stampexpiry`
  check yet (PR #591 is external); `pkg/check/gc/reserve.go` is the tiny-amount-batch precedent.
- **Redistribution round-loss — assertable WITHOUT `/metrics` scraping.** Checks can't scrape
  Prometheus, but `/redistributionstate` already returns `Round`, `LastPlayedRound`, `LastWonRound`,
  `LastSelectedRound`, `LastSampleDurationSeconds`, `IsFullySynced`, `IsFrozen`. Round-loss =
  `Round` advances while `LastPlayedRound`/`LastWonRound` stalls (and `IsFullySynced=false`). The
  check reads `/redistributionstate` today but ignores these fields.
- **Not available:** the `bee_pullsync_chunks_delivered` "delivered-plateau" smoking gun (needs a new
  `/metrics` scrape on the bee client). The structured equivalents — `pullsyncRate` decay +
  `reserveSizeWithinRadius` plateau + `fullySynced` stuck false — are enough; treat `/metrics`
  scraping as a deferred nice-to-have (Phase 7).

## Design

A new self-driving **`mode: halt`** that composes the existing pieces plus two new stages, so one
run reproduces the whole recipe:

```
stake(optional) → waitWarmup → driveIncrease(to DisruptAtRadius) → settle
  → disrupt(node-churn | batch-expiry) → observeOutcome(on survivors)
```

The disruption + staking + outcome-classification are written as **composable stages gated by
options**, so the parallel-with-load shape reuses them: `mode: observe` + a disruption mechanism means
"don't upload — let the `load` check drive the radius — but still stake, disrupt at the target, and
classify the outcome."

**Both shapes are first-class (decided).** Write the stages composable so the radius can be driven
either by the check itself (`mode: halt`) **or** by a parallel `load` check (`mode: observe` +
disrupt). To avoid a fragile cross-check coordination channel, the observer **tolerates ongoing
uploads**: with `node-churn` the halt is a neighbourhood-redundancy drop, not a committed-depth effect,
so the assertions (`fullySynced`, round-loss) hold whether or not `load` keeps running. The manual run
*stopped* load only for a cleaner CSV; it is not required for the node-churn halt. **Caveat for
`batch-expiry`:** that lever works by *dropping* commitment, so ongoing `load` (which re-adds
commitment) counteracts it — in parallel-with-load, prefer `node-churn`, or have `load` wind down
around the expiry. Single-mode remains the most faithful reproduction; parallel-mode is the
soak/realism shape.

**Disruption mechanisms (pluggable: node-churn and/or batch-expiry).** `disrupt-mechanism` chooses how
the neighbourhood is disrupted, both feeding the same observe/classify/verdict stage:
- `node-churn` (default) — stop/delete randomly-chosen full nodes. The reliable redundancy-drop lever
  (what the manual repro used): removes neighbourhood replicas so survivors must re-replicate.
- `batch-expiry` — the **original October mechanism**. Create a short-lived postage batch, fill the
  reserve under it, then let it expire: bee evicts its chunks cluster-wide (`evictExpiredBatches`),
  commitment drops, and on a synced node below the decrease threshold the **storage radius decreases**
  → neighbourhood merge → pull-sync re-sync. More faithful to October than node-churn, but the
  decrease is **gated** (`SyncRate==0` + `countWithinRadius < threshold`) so it fires intermittently —
  the `radius_threshold.patch` (threshold = capacity) makes it reachable on a small cluster.
- `both` — expire a batch *and* churn nodes for a harsher merge; `none` — monitor-only.

**Tunable disruption + a non-failing verdict (decided).** Each mechanism's intensity is tunable
(`disrupt-node-count` for node-churn, **randomly selected** seeded by `rnd-seed`; `expiring-batch-ttl`
for batch-expiry) and the whole disruption can be turned off (`disrupt-mechanism: none`, or
`disrupt-node-count: 0`) so the check just monitors radius / sync / round state for `duration` (watch a
healthy cluster, or build a baseline). Either way the check **reports the outcome rather than failing
on it** — so the *same* check can reproduce the halt, deliberately *not* reproduce it (low intensity),
or just monitor, all without a red result.

**Outcomes** the observe stage classifies and emits (a metric + an end-of-run summary line):
- `MONITORED` — `disrupt-node-count: 0`; no disruption, state recorded over `duration`.
- `HALT` — after disruption, sustained non-convergence (survivors stuck `fullySynced=false` past a
  bound) and/or staked round-loss.
- `RECOVERED` — after disruption, all survivors returned to `fullySynced` (and resumed playing rounds)
  within `recovery-wait`.

**Verdict policy** (`verdict`):
- `report` (**default**) — always return success on any observed outcome; emit the outcome + metrics +
  a clear summary. Fail **only** on operational errors (warmup timeout, stake/removal failure, all
  nodes unreachable). This is the explore / soak / reproduce-or-not mode.
- `assert` — gate on the expectation: fail iff the observed outcome contradicts `expect-recovery`
  (`false` ⇒ expect `HALT`, `true` ⇒ expect `RECOVERED`). For the A/B regression gate. `MONITORED`
  (no disruption requested) is always a pass.

## Phases

### Phase 0 — Shape decisions (RESOLVED)
- [x] **Both shapes first-class**: `mode: halt` (self-driving) **and** `mode: observe` + disrupt
      (parallel with `load`). Stages written composable to serve both.
- [x] **Parallel contract**: observer **tolerates ongoing uploads** (no cross-check IPC); assertions
      are upload-robust. Stopping `load` is optional, not required.
- [x] **Node-removal method default = `stop`** (scale statefulset to 0; restorable; emptyDir means a
      restart is a fresh node so restore-mid-run needs re-funding — a Phase-7 nicety). `delete` stays
      available as an option.
- [x] **A-side semantics = PASS when halt reproduced** (`expect-recovery: false` is the default →
      reproduction gate, not a regression detector).
- [x] **Tunable disruption + non-failing default**: `disrupt-node-count` randomly selects nodes
      (`0` = monitor-only), and `verdict: report` (default) records `HALT`/`RECOVERED`/`MONITORED`
      and only fails on operational errors. `verdict: assert` opts into A/B gating via `expect-recovery`.

### Phase 1 — Staking pre-step
- [x] Add options: `StakeAmount` (config `stake-amount` string→parsed `*big.Int` in the stage,
      e.g. `"100000000000000000"`; empty/`"0"` = skip), `StakeGroups` (`stake-groups`, default =
      observed groups). Held as a `string` in `Options` (parsed at use-time) so config mapping and the
      empty=skip sentinel stay trivial.
- [x] New stage `ensureStaked` (+`ensureNodeStaked`): for each node in the staking set (`StakeGroups`,
      or the observed nodes), `GetStake`; if `< StakeAmount`, `DepositStake`; re-read to verify.
      Idempotent (already-at-target nodes skipped), logged per node. Wired before warmup in both
      `runDrive` and `runObserve`, gated on `stake-amount` set.
- [x] Budget the ~10-block usable wait via `waitStakeAtLeast` + `stake-confirm-wait` (default `2m`):
      `POST /stake` returns a tx hash, so poll `GetStake` until the deposit confirms on-chain (or the
      budget expires). Already-mined deposits pass immediately.

### Phase 2 — Drive (or wait) to the disruption radius
- [x] Add `DisruptAtRadius` (default 3, `disrupt-at-radius`) distinct from `TargetRadius`.
- [x] `mode: halt` (`runHalt`): stake → warmup → `driveAllToRadius` (gates on the **min** node radius,
      not the max, so ALL observed nodes reach `DisruptAtRadius`) → settle window. `updatePeak` now
      returns `(min, max)`. Disruption (Phase 3) + outcome observation (Phase 4) are stubbed with TODOs.
- [ ] `mode: observe`+disrupt: poll until every observed node reports
      `storageRadius >= DisruptAtRadius` (load drives), with a timeout.
      **(Resequenced → Phase 3: the wait-to-radius helper is dead code until the observe+disrupt
      dispatch exists, which needs the Phase-3 `disrupt-mechanism`/`disrupt-node-count` options.)**
- [x] Record a pre-disruption baseline snapshot (`baselineSnapshot`): per node, storageRadius +
      reserveSizeWithinRadius (/status), isFullySynced + round/lastPlayed/lastWon
      (/redistributionstate), and stake — emitted + logged. Wired into `runHalt` (replaces the plain
      `snapshot("baseline")`). Phase 4 will extend it to return the values for onset/round-loss compare.

### Phase 3 — Disruption mechanisms (`disrupt-mechanism`)
Shared: a `disruption_total` metric + timestamp marks the onset reference. `disrupt-mechanism: none`
(or node-churn with `disrupt-node-count: 0`) skips straight to observe (monitor-only).

**3.0 — observe+disrupt dispatch + staging** (resequenced from Phase 2)
> **Deferred until 3a (disrupt stage) + Phase 4 (observe-outcome) land.** The dispatch wires together
> stages that don't exist yet, and `runHalt` is the ready consumer for those stages — so build the
> mechanism + observe-outcome first, then add this dispatch to reuse them. Working 3a first.
- [ ] Add `disrupt-mechanism` dispatch so `mode: observe` with a mechanism set runs the staged
      reproduction (wait-to-radius → baseline → disrupt → observe) instead of the plain soak monitor.
- [ ] `mode: observe`+disrupt staging: poll until every observed node reports
      `storageRadius >= DisruptAtRadius` (load drives), with a timeout; then `baselineSnapshot`.

**3a — node-churn**
- [x] Add the disruption option surface: `DisruptMechanism` (`disrupt-mechanism`, default `node-churn`),
      `DisruptNodeCount` (default 2; `0` = skip), `DisruptMethod` (`stop` default | `delete`), and
      `MinSurvivors` (default 3) — with mechanism (`DisruptNodeChurn`/`DisruptBatchExpiry`/`DisruptBoth`/
      `DisruptNone`) and method (`RemoveStop`/`RemoveDelete`) constants, config mapping, defaults, README.
      *(MinSurvivors + DisruptMechanism folded in here since the node-churn stage consumes all of them.)*
- [x] **Randomly** select `DisruptNodeCount` nodes, seeded by `rnd-seed` (`rnd.Perm`, reproducible);
      excludes `excludeName` (the uploader in `halt` mode). Builds the **survivor set** (observed minus
      removed) returned for the observe loop. `DisruptNodeCount <= 0` = monitor-only (returns all).
- [x] Remove via `node.Stop(ctx, namespace)` / `node.Delete(...)` (looked up from `cluster.Nodes()` by
      name), selected by `DisruptMethod`. Logs + RFC3339-timestamps each removal; emits `disruption_total`
      (labelled by mechanism). Implemented as `disruptNodeChurn`, behind a `disrupt` mechanism dispatcher
      (node-churn/none done; batch-expiry/both error as Phase-3b). Wired into `runHalt`.
- [x] Guard: `len(observed) - DisruptNodeCount >= MinSurvivors`, plus a candidate-count check — both
      error out **before** any removal touches the cluster.

**3b — batch-expiry** *(deferred until after Phase 4)*
> **Deferred:** its defining items (detect expiry → observe eviction → gated radius decrease, else
> `MONITORED`) are observe-phase logic that must fold into the Phase-4 observe loop, which doesn't
> exist yet. Node-churn (the primary mechanism the manual repro used) is fully wired, so build Phase 4
> against it first, then add batch-expiry with its detection folded into the existing loop.
- [ ] Create a soon-to-expire mutable batch — `CreatePostageBatch(ctx, amount, depth, label)` with a
      small explicit `amount`, or `GetOrCreateMutableBatch` with a short `expiring-batch-ttl`. Drive the
      radius (Phase 2) by uploading the fill **under this batch** so its chunks dominate the reserve.
- [ ] Detect expiry: poll `PostageStamp(batchID)` for `BatchTTL`→0 / `Exists=false`, cross-checking
      `GetChainState().TotalAmount` against the batch `Amount`; timestamp it as onset.
- [ ] Observe eviction: `reserveSize` / `reserveSizeWithinRadius` drop as `evictExpiredBatches` runs;
      then watch for the **gated** radius decrease (synced + `count<threshold`) → merge → re-sync. If
      the gate never trips (no decrease), classify `MONITORED` with eviction recorded — don't fail.
- [ ] No node removal — all nodes stay in the observe set.

### Phase 4 — Observe, classify, report (optionally assert) + round-loss
- [x] Extend the per-node observe state (`outcomeNode` in `observeOutcome`): track **onset**
      (`fullySynced` true→false relative to the first post-disruption reference), **recovery** (back to
      `fullySynced` within `RecoveryWait`), and **round participation** (`Round` advancing while
      `LastPlayedRound`/`LastWonRound` stall = staked round-loss).
- [x] Classify the run outcome → `MONITORED` | `HALT` | `RECOVERED` (`classifyOutcome`); emit the
      `outcome` gauge (one-hot) + a clear end-of-run summary (stuck vs recovered survivors, round-loss).
- [x] New metrics: `outcome` (labelled), `disruption_total`, `onset_seconds{node}` (disrupt→onset, set
      on onset) and `round_loss_total{node}` (incremented per round-loss survivor), reusing
      `fully_synced`/`frozen`/`redistribution_round`/`reserve_within_radius`/`pullsync_rate`/
      `time_to_fully_synced_seconds`.
- [x] Apply the **verdict policy** (`applyVerdict`, `verdict`/`expect-recovery` options + constants):
      - `report` (default) → always return success on the outcome; only operational errors fail.
      - `assert` → fail iff the outcome contradicts `expect-recovery`; `MONITORED` always passes.
- [x] Bound the whole observe by `Duration` (`observeOutcome` deadline; onset is delayed/variable ~3–9 min).

### Phase 5 — Wire options, config, docs
- [ ] Add the new fields to `pkg/config/check.go` `"reserve-radius"` `NewOptions` struct + defaults in
      `NewDefaultOptions`.
- [ ] Add config entries to `config/local.yaml`: `ci-radius-halt` (single self-driving) and, if Phase 0
      keeps it, a parallel pair reusing `ci-load-soak` + `ci-reserve-radius-observe`+disrupt.
- [ ] Update `pkg/check/reserveradius/README.md`, the `radius-testing` skill, and cross-link
      `docs/radius-halt.md` ↔ this plan.

### Phase 6 — A/B validation
- [ ] Run `ci-radius-halt` against **A** (patched stock master) → expect halt reproduced
      (`expect-recovery: false` passes; matches the manual run's numbers).
- [ ] Exercise **both mechanisms** on A: `node-churn` (matches the manual repro) and `batch-expiry`
      (the commitment-drop → radius-decrease → merge path); confirm each reaches `HALT` or, when the
      decrease gate doesn't trip, `MONITORED` (not a failure).
- [ ] Build **B** (`pullsync-optimal-design` + the same `radius_*.patch`) and run with
      `expect-recovery: true` → expect survivors re-converge.
- [ ] `make build vet lint test-race` green. No unit tests required for the check (per prior decision);
      validate live on `local-dns`.

### Phase 7 — Deferred: real signals & scale
- [ ] Optional `/metrics` scrape on the bee client to capture the `chunks_delivered` plateau and
      `bee_storageincentives_is_playing_errors`/`winner` directly (richer round-loss proof).
- [ ] Run unpatched on a ~20-node ephemeral cluster (the merge needs scale); periodic/RC cadence.

## New options (summary)

| Option | Default | Purpose |
| --- | --- | --- |
| `mode` | `drive` | add `halt` (self-driving stake→drive→disrupt→observe) |
| `stake-amount` | `""` (skip) | per-node stake to ensure before driving (e.g. `100000000000000000`) |
| `disrupt-mechanism` | `node-churn` | `node-churn`, `batch-expiry`, `both`, or `none` (monitor-only) |
| `disrupt-at-radius` | `3` | radius all observed nodes must reach before disruption |
| `disrupt-node-count` | `2` | node-churn: full nodes to remove, **randomly** (seeded); `0` = none |
| `disrupt-method` | `stop` | node-churn: `stop` (scale-0, default) or `delete` (statefulset+resources) |
| `expiring-batch-ttl` | `0` | batch-expiry: TTL for the short-lived fill batch (0 = use a small explicit amount) |
| `min-survivors` | `3` | refuse to disrupt below this |
| `verdict` | `report` | `report` (never fail on outcome; only operational errors) or `assert` (gate on `expect-recovery`) |
| `expect-recovery` | `false` | **`assert` mode only**: expected outcome (false=`HALT`, true=`RECOVERED`) |
| `recovery-wait` | `10m` | per-survivor convergence bound after onset |
| `duration` | `30m` | total observe window (onset is delayed ~3–9 min) |

## Decisions made (Phase 0)

- Both shapes first-class (single `halt` + parallel `observe`+disrupt); observer tolerates ongoing load.
- Disruption is a **pluggable mechanism** — `node-churn` (default), `batch-expiry` (the October
  commitment-drop → radius-decrease lever), `both`, or `none`. `disrupt-node-count` is tunable and
  **randomly** selects nodes; any mechanism can be turned off for monitor-only.
- Default `verdict: report` → the check **records the outcome (`HALT`/`RECOVERED`/`MONITORED`) and
  does not fail** whether the halt reproduces, doesn't, or no disruption was requested. Only
  operational errors fail. `verdict: assert` + `expect-recovery` opts into the A/B regression gate.
- Node removal default = `stop` (scale-0); `delete` available as an option.

## Remaining open question

- **`/metrics` scraping** (Phase 7): add a `/metrics` scrape on the bee client now for the
  `chunks_delivered` plateau + `bee_storageincentives_is_playing_errors`/`winner` (richer round-loss
  proof), or ship v1 on structured `/redistributionstate` signals only? Plan currently **defers** it.

## References

- `docs/radius-halt.md` — the reproduced recipe + measured numbers (mechanism, steps, round-loss, A/B).
- `docs/radius-check-plan.md` — the original check design (Phases 0–2.5 done).
- `pkg/check/reserveradius/` — current check (drive/observe modes).
- `pkg/check/stake/stake.go` — staking-API usage precedent.
- `pkg/orchestration/{node,nodegroup,cluster}.go` + `k8s/orchestrator.go` — node Delete/Stop.
- `pkg/bee/api/postage.go` + `client.go` (`CreatePostageBatch`, `GetOrCreateMutableBatch`,
  `PostageStamp`, `GetChainState`) — batch creation + expiry/chainstate reads.
- bee `pkg/postage/batchstore/store.go` (`cleanup`→`evictFn`) + `pkg/storer/reserve.go`
  (`evictExpiredBatches` → gated radius decrease) — the expiry→eviction→decrease path.
- `.claude/skills/radius-testing/` — operating know-how + `scripts/radius-poll.sh`.

# reserve-radius check

Exercises a **storage-radius change** across a Bee cluster and verifies the node
recovers afterwards — catching regressions in the puller's reaction to a radius
change (the `manage()` / `disconnectPeer()` path that caused the liveness bug in
beekeeper PR #581).

It has three modes (`Options.Mode`):

- **`drive`** (default) — a one-shot run: upload to force the radius up, then watch
  it come back down and assert pull-sync recovers.
- **`observe`** — a long-running monitor: does **not** upload; watches radius
  transitions for `Duration` while something else drives the cluster (typically the
  `load` check running in parallel via `--parallel`), recording every up/down
  transition and asserting recovery after each decrease. This is the soak mode.
- **`halt`** — self-driving pull-sync-halt reproduction: stake → drive **all** observed
  nodes to `DisruptAtRadius` → settle → disrupt the neighbourhood → observe the outcome.
  Disruption mechanisms and outcome classification are being built out (Phases 3–4);
  currently it runs the stake/drive/settle prefix.

> Keep this file in sync with the code. If you change `Options`, the `Run` flow, the
> emitted metrics, the registration, or the bee-patch requirement, update the matching
> section below in the same change.

## What it does (`Run` flow)

`Run` dispatches on `Mode`. Both modes first run an optional **staking pre-step**.

**staking pre-step** (`ensureStaked`, both modes): when `stake-amount` is set, ensure every node in
the staking set (`stake-groups`, or the observed nodes) has at least that stake — `GetStake`, and if
below, `DepositStake` then poll until the deposit confirms on-chain (within `stake-confirm-wait`).
Idempotent: nodes already at/above target are skipped. Runs before warmup so deposits confirm while
the cluster stabilizes.

**`drive`** (`runDrive`):

1. **`waitForWarmupDone`** — block until every observed node reports
   `isWarmingUp == false`. The reserve worker's *decrease* loop is gated on
   stabilization, so this is a precondition for the radius ever dropping.
2. **baseline** — snapshot `/status` for all observed nodes.
3. **`driveIncrease`** — buy a mutable batch on a random full node, then upload
   `BlobSize` random blobs to it until any observed node's `storageRadius` reaches
   `TargetRadius` (or `MaxUploads` / `IncreaseTimeout`). Tracks a **per-node
   high-water `peak`**.
4. **settle** — keep polling for `SettleWait`, still raising `peak` (on an
   already-stabilized node the decrease can begin *during* settle, so `peak` must be
   the max seen, not a single post-settle read).
5. **`observeDecrease`** — poll until any node's `storageRadius` falls below its
   `peak`, watching `pullsyncRate` for recovery. No decrease within `DecreaseTimeout`
   fails with a message pointing at the PR #581 puller stall.

**`observe`** (`runObserve`): `waitForWarmupDone`, then for `Duration` poll every
`PollInterval`, reading both `/status` and **`/redistributionstate`** per node. It records
up/down `storageRadius` transitions, and asserts the **halt indicators** that define the
October failure:

- **recovery after a decrease** — after a **down** transition, the node must recover within
  `RecoveryWait`. Recovery prefers the redistribution signal **`isFullySynced && !isFrozen`**
  (the real "can play the game" signal); if `/redistributionstate` is unavailable (not
  full-mode), it falls back to `pullsyncRate > 0`.
- **freeze episodes** — a node reporting `isFrozen` is skipping redistribution rounds (a halt
  symptom); each freeze episode is counted.

Un-recovered decreases or freezes fail the check at the end. Never uploads (the radius is
driven externally, e.g. by a parallel `load` check).

**`halt`** (`runHalt`): the self-driving reproduction.

1. **stake** (optional `ensureStaked` pre-step, as above).
2. **`waitForWarmupDone`** + **`baselineSnapshot`** — the pre-disruption reference: per node, storage
   radius + `reserveSizeWithinRadius`, `isFullySynced` + round/lastPlayed/lastWon, and stake.
3. **`driveAllToRadius`** — buy a mutable batch and upload until **every** observed node's
   `storageRadius` reaches `DisruptAtRadius` (gates on the **min** across nodes, so the whole
   neighbourhood is populated before disruption), or `MaxUploads` / `IncreaseTimeout`.
4. **settle** — poll for `SettleWait`.
5. **`disrupt`** — apply `DisruptMechanism`. **node-churn** (`disruptNodeChurn`): randomly pick
   `DisruptNodeCount` nodes (seeded by `rnd-seed`, excluding the uploader), guard `MinSurvivors`
   **before** touching the cluster, then `Stop` (scale-0) or `Delete` each; returns the **survivor set**.
   `none` / `disrupt-node-count: 0` is monitor-only; `batch-expiry`/`both` are Phase 3b (not yet).
6. **`observeOutcome`** — poll the survivors for `Duration`, tracking per node the **onset** of de-sync
   (`isFullySynced` true→false), **recovery** (back within `RecoveryWait`), and staked **round-loss**
   (`round` advancing while `lastPlayed`/`lastWon` stall). `classifyOutcome` reduces this to
   `MONITORED` (no disruption) / `HALT` (a survivor stuck de-synced past `RecoveryWait` and/or
   round-loss) / `RECOVERED` (all de-synced survivors re-converged), emitting the `outcome` gauge.
   The **verdict** (report-vs-assert) is the next step (currently always reports).

## Options

Defaults in `NewDefaultOptions()`; YAML keys (kebab-case) are wired in
`pkg/config/check.go` under the `reserve-radius` entry.

| field | yaml | default | mode | purpose |
| --- | --- | --- | --- | --- |
| `Mode` | `mode` | `drive` | all | `drive` (force a change), `observe` (monitor only), or `halt` (self-driving reproduction) |
| `Duration` | `duration` | `12h` | observe | total monitor run length |
| `RecoveryWait` | `recovery-wait` | `5m` | observe | max wait for pull-sync recovery after each decrease |
| `RndSeed` | `rnd-seed` | `time.Now().UnixNano()` | both | seed for `random.PseudoGenerator` → shuffled node pick |
| `StakeAmount` | `stake-amount` | `""` (skip) | both | per-node stake (wei) to ensure before driving, e.g. `"100000000000000000"`; empty/`"0"` skips |
| `StakeGroups` | `stake-groups` | `nil` (observed) | both | node groups to stake (empty = the observed/selected groups) |
| `StakeConfirmWait` | `stake-confirm-wait` | `2m` | both | max wait for a deposit to confirm on-chain (~10 blocks) |
| `UploadGroups` | `upload-groups` | `[bee]` | both | node groups to observe (and, in drive, upload to) |
| `PollInterval` | `poll-interval` | `15s` | both | poll cadence |
| `PostageTTL` | `postage-ttl` | `24h` | drive | batch TTL (use TTL, not a raw amount) |
| `PostageDepth` | `postage-depth` | `22` | drive | batch depth |
| `PostageLabel` | `postage-label` | `reserve-radius` | drive | batch label |
| `BlobSize` | `blob-size` | `1048576` (1 MiB) | drive | bytes per upload |
| `MaxUploads` | `max-uploads` | `60` | drive | cap on uploads in the increase phase |
| `TargetRadius` | `target-radius` | `1` | drive | storageRadius any node must reach before stopping uploads |
| `DisruptAtRadius` | `disrupt-at-radius` | `3` | halt | storageRadius **all** observed nodes must reach before disruption |
| `DisruptMechanism` | `disrupt-mechanism` | `node-churn` | halt/observe+disrupt | `node-churn`, `batch-expiry`, `both`, or `none` (monitor-only) |
| `DisruptNodeCount` | `disrupt-node-count` | `2` | halt/observe+disrupt | node-churn: full nodes to remove (randomly, seeded by `rnd-seed`); `0` = skip |
| `DisruptMethod` | `disrupt-method` | `stop` | halt/observe+disrupt | node-churn: `stop` (scale statefulset to 0) or `delete` (statefulset + resources) |
| `MinSurvivors` | `min-survivors` | `3` | halt/observe+disrupt | refuse to disrupt below this many surviving nodes |
| `WarmupWait` | `warmup-wait` | `15m` | both | max wait for nodes to finish warmup |
| `IncreaseTimeout` | `increase-timeout` | `5m` | drive | max time to reach `TargetRadius` |
| `SettleWait` | `settle-wait` | `1m` | drive | post-upload window (pushsync drain + peak tracking) |
| `DecreaseTimeout` | `decrease-timeout` | `20m` | drive | max time to observe a decrease |

## Metrics (`metrics.go`)

Emitted via the `metrics.Reporter` interface (pushed to the pushgateway when the
check runs with `--metrics-enabled`). Namespace `beekeeper`, subsystem
`check_reserve_radius`:

- `…_storage_radius{node}` — gauge, storageRadius per node
- `…_reserve_size{node}` — gauge, reserve size (chunks) per node
- `…_pullsync_rate{node}` — gauge, `/status` pullsyncRate per node
- `…_time_to_increase_seconds` / `…_time_to_decrease_seconds` — gauges (drive mode)
- `…_reserve_within_radius{node}` — gauge, reserve chunks within radius (completeness signal)
- `…_radius_transitions_total{node,direction}` — counter, observed up/down transitions (observe mode)
- `…_recovery_observed_total{node,result}` — counter, recovery outcome `recovered`/`timeout` (observe mode)
- `…_disruption_total{mechanism}` — counter, neighbourhood disruptions applied (halt mode, e.g. `node-churn`)
- `…_outcome{outcome}` — gauge, one-hot halt-run classification (`MONITORED`/`HALT`/`RECOVERED`); the classified one is 1
- `…_time_to_fully_synced_seconds` — gauge, decrease → isFullySynced again (observe mode)
- from `/redistributionstate` (observe mode): `…_fully_synced{node}`, `…_frozen{node}`,
  `…_redistribution_round{node}`, `…_last_sample_duration_seconds{node}` — the halt indicators

## Requirements

- **Patched bee image.** On stock capacity the reserve is far too large to move.
  The check needs a node built with `bee/.github/patches/radius_reserve.patch`
  (`DefaultReserveCapacity`→200, `ReserveWakeUpDuration`→10s) and
  `radius_threshold.patch` (decrease `threshold`→100%). See
  `docs/radius-check-plan.md` and the `radius-testing` skill.
- **`isWarmingUp` in `StatusResponse`** (`pkg/bee/api/status.go`) — the node returns
  it; the struct field was added for this check's warmup gate.

## Running it

Against a patched local cluster (`local-dns`):

```sh
# drive (one-shot): force a change and assert recovery
./dist/beekeeper check --cluster-name=local-dns --checks=ci-reserve-radius --log-verbosity=info

# soak: load oscillates the radius, reserve-radius observes it, both run concurrently
./dist/beekeeper check --cluster-name=local-dns \
  --checks=ci-load-soak,ci-reserve-radius-observe --parallel \
  --metrics-enabled=true --metrics-pusher-address=localhost:9091 --log-verbosity=info
```

`--parallel` runs the listed checks in goroutines instead of sequentially;
checks fail independently (a monitor failure does not stop the load run).

## Observed behavior (local, patched cluster)

- **Increase is fast** — ~1 MiB moves `storageRadius` 0→1 in seconds.
- **Decrease lags ~13 min** and is stabilization-gated; sized by `DecreaseTimeout` (20m).
- **No fixed floor** — the equilibrium radius depends on data volume (a data-heavy
  node can stay elevated and never decrease in-window). The check asserts *a decrease
  occurred*, not that the radius returns to a specific value.
- **Pull-sync on a radius change is a puller reconfiguration**: `bee_puller_worker`
  contracts on increase / expands on decrease, with a `bee_puller_worker_errors`
  cancel-storm. Actual resync volume (`bee_pullsync_chunks_delivered`) is ~0 on a
  near-empty cluster — meaningful resync needs data (real cluster).

## Known limitations / follow-ups

- Observe-mode recovery uses `/redistributionstate` (`isFullySynced`/`isFrozen`) where
  available, falling back to `/status` `pullsyncRate` (weak on light clusters — can
  false-`timeout`). The **exact thresholds** (the `isFullySynced` bound, acceptable
  convergence time, the `offered/delivered` redundancy target) need to come from the
  pull-sync design owner (@marios) — see `docs/radius-check-plan.md` "Pull-sync validation".
- Per-`(peer,bin)` `STALLED` / `MaxStallsPerBin` signals from the **fixed** pull-sync
  (@sig's PR) are not yet scraped (they don't exist on stock bee).
- The neighbourhood **merge** (`k→8` on a decrease) needs a ~20-node cluster to reproduce;
  the local 3-node cluster proves the mechanism and pipeline only.
- Direction is fixed to increase-then-decrease (drive) / observe-both (observe); no
  `increase`-only assertion mode yet.
- No unit tests yet (external `_test` package TBD).

## Related

- `docs/radius-check-plan.md` — design, phases, spike findings, the driver/observer split.
- `.claude/skills/radius-testing/` — operating know-how, `radius-poll.sh`, the
  Pushgateway/Prometheus/Grafana metrics stack (`metrics/`).
- Prior art: PR #581 (`radiusdecrease`), PR #591 (`stampexpiry`), `pkg/check/gc`,
  `pkg/check/load` (the committedDepth-gated upload primitive, reused as the soak driver),
  `pkg/check/smoke`.

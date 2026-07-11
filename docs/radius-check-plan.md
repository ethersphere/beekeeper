# Reserve-radius change check — plan

Status: **Local check done (single idle-driven flow); Phase 3 + A/B pending.** Branch `ljubisa-radius-soak-v2`.

A repeatable, automated way to stage a **storage-radius change** across a Bee cluster and
verify pull-sync recovers afterwards. End product: a beekeeper check
(`pkg/check/reserveradius`) plus the metrics it emits. Operating know-how lives in the
`radius-testing` skill; this doc is the design and build checklist.

## Pull-sync validation (the real target)

The narrow goal is "force a radius decrease, confirm the puller doesn't deadlock" (PR #581).
The broader goal — why @sig/@marios want this — is a **reproducible test of the October
network halt** and validation of the pull-sync redesign (SWIP-25 / `pullsync-optimal-design`).

**The October mechanism.** A large operator went offline → commitment dropped → **storage
radius decreased**, a neighbourhood **merge**: `k` jumps to ~8, so every node must re-sync a
much larger reserve from many more peers. Today's pull-sync is "correctness through
redundancy" — up to **k× redundant deliveries** per chunk, no shared dedup, no
failover-with-exclude; on a depth change bins are cancelled/restarted repeatedly (unbounded
delay) and stalls livelock. Result: incomplete reserves → wrong `ReserveSample` → nodes
**freeze, skip rounds, go unresponsive** (slashing risk).

**What the test must show** on a staged decrease:

- **Completeness** — reserves catch up (`reserveSizeWithinRadius` recovers).
- **Liveness** — no freezes, `round` keeps advancing, `isFullySynced` returns within a bound.
- **Bounded convergence** — resync finishes in bounded time; no stuck `STALLED` bins.
- **Efficiency (SWIP-25)** — redundant deliveries drop ~k×→1× (`offered/delivered` ratio).

**Implemented so far:** a single self-driving check (`pkg/check/reserveradius`) drives the radius
up by uploading random data to random nodes to a target (≥3), stops, waits for pull-sync to go idle
(`pullsyncRate<=0.05` — the reserve worker only decreases while `SyncRate()==0`), dilutes one batch
to the chain's min-validity floor to hasten its expiry (force-decrease), then asserts the radius
ticks back down below its peak within a timeout. Metrics per node: `storage_radius`, `reserve_size`,
`reserve_within_radius`, `pullsync_rate`, plus `time_to_increase_seconds` / `time_to_decrease_seconds`
/ `dilution_total`. Validated live 2026-07-11 on the patched 6-node local-dns: 0→3 in ~64 MiB /
~6 min, batch diluted 48h→6h (floored), radius 3→2 at ~8 min. **Locally the decrease is pull-sync-
idle-gated** (threshold patched to 100% ⇒ commitment/expiry irrelevant); **on testnet it is
expiry-driven** — dilution hastens the batch expiry that drops commitment. The deeper liveness
signals (`/redistributionstate` freeze / round-loss)
are documented in the `radius-testing` skill for follow-up.

**Still needs @marios + scale:** exact pass/fail thresholds; the per-`(peer,bin)` `STALLED` /
`MaxStallsPerBin` metrics (only @sig's fixed PR exposes them); the **~20-node cluster** (the
merge needs scale); the **A/B harness** (below).

## Prior art

- **PR #581** (`radiusdecrease`) — forces a decrease (overflow tiny reserve, 0→1→0), polls
  `/status` for `pullsyncRate>0` recovery. Needs a patched bee image. The real radius tryout.
- **PR #591** (`stampexpiry`) — patch-free, observes radius via stamp expiry. Review feedback:
  use `PostageTTL`, random node, periodic/RC cadence, not standard CI.
- **GC check** (`pkg/check/gc/reserve.go`) — precedent for a patch-dependent check; patches live
  in `bee/.github/patches/`, applied by bee's beekeeper workflow.

## Signals

Per node: `/status` (`storageRadius`, `pullsyncRate`, `reserveSize`, `reserveSizeWithinRadius`,
`committedDepth`, `isWarmingUp`), `/reservestate` (network `radius`), `/redistributionstate`
(`isFullySynced`, `isFrozen`, `round`). Prometheus: `bee_localstore_*`, `bee_pullsync_*`,
`bee_storageincentives_*`. Full table in the `radius-testing` skill.

## Approach

Patched-local first (deterministic, fast transitions) → unpatched ephemeral k8s (realistic
timing, periodic/RC cadence). Same check, different environment and timeouts.

## Phase 0 — Local scaffolding (done)

- [x] `/cluster-verify`, `/cluster-up`, `/cluster-down` commands.
- [x] `radius-testing` skill + `scripts/radius-poll.sh` monitor.

## Staging the change (reuse the `load` check)

Don't pre-compute byte counts — upload in a loop, watch the signal, stop at a target. `load`
(`pkg/check/load/load.go`) already does this: `MaxCommittedDepth` gates uploads via
`checkCommittedDepth` (`committedDepth = storageRadius + capacityDoubling`).

**Increase and decrease are not symmetric.** Increase is upload-driven (fill past capacity →
eviction → radius up). Decrease is not — per `bee/pkg/storer/reserve.go`, radius drops only when
the reserve is under-utilized AND fully synced (`SyncRate()==0`) above `minimumRadius`. Trigger
it by inflating first, then letting the patched reserve worker re-evaluate (0→1→0 cascade).

## Phase 1 — Spike: force a change with a patched bee (done)

- [x] **Patch created** in `bee/.github/patches/` (GC-check pattern; all three symbols are
      unreachable by config) — what and why:
      - `radius_reserve.patch`: `DefaultReserveCapacity` 1<<22→**200** (so ~2 MB moves the radius —
        stock is too large to budge in CI-time) and `ReserveWakeUpDuration` 30m→**10s** (the reserve
        worker re-evaluates in seconds, not 30 min).
      - `radius_threshold.patch`: decrease `threshold` 50%→**100%** (makes the 1→0 decrease reachable).
      gofmt-clean, applies cleanly.
- [ ] **Apply, build, push, redeploy** (revert source after build):

```sh
cd <bee-repo>
patch pkg/storer/reserve.go .github/patches/radius_threshold.patch
patch pkg/storer/storer.go  .github/patches/radius_reserve.patch
make docker-build PLATFORM=linux/arm64 BEE_IMAGE=k3d-registry.localhost:5000/ethersphere/bee:latest \
  REACHABILITY_OVERRIDE_PUBLIC=true BATCHFACTOR_OVERRIDE_PUBLIC=2
docker push k3d-registry.localhost:5000/ethersphere/bee:latest
git checkout pkg/storer/reserve.go pkg/storer/storer.go
```

Then `/cluster-down local-dns` → `/cluster-up local-dns`. To wire into CI, add the two `patch`
lines to bee's `.github/workflows/beekeeper.yml` "Apply patches and build" step.

### Why the check is shaped this way (spike, 2026-06-17)

The decrease is slow and non-obvious; each fact below maps to a check decision:

- **Lags ~10 min, stabilization-gated** (the reserve worker's decrease loop waits on the
  startup-stabilizer) → long timeouts (decrease ≥15 min), gate on `isWarmingUp==false`.
- **Overshoots, then settles in a staircase** (in-flight pushsync keeps filling after uploads stop;
  each decrease triggers pull-sync that blocks the next) → wait for pushsync drain and track the
  high-water peak before treating a value as settled.
- **No fixed floor** (radius held at 1 for 30+ min) → assert *"a decrease occurred AND pull-sync
  recovered"*, not *"radius returned to N"*.
- **Puller drives it** — `node/puller "radius decrease"` + a `syncWorker context cancelled` storm =
  the PR #581 `manage()`/`disconnectPeer()` path (the liveness risk).

## Phase 2 — The check (done)

`pkg/check/reserveradius/`. `Run` = `selectNodes` → `waitForWarmupDone` → baseline →
`driveIncrease` (upload random data to random nodes under this check's own postage label, to
`target-radius`) → `waitPullSyncIdle` (wait `pullsyncRate<=0.05` on all nodes) →
`diluteToMinValidity` (force-decrease: dilute one batch to the min-validity floor to hasten expiry)
→ `observeDecrease` (assert the radius ticks below its peak within `decrease-timeout`). Registered
as type `reserve-radius` (`pkg/config/check.go`); `ci-reserve-radius` in `config/local.yaml`; added
`IsWarmingUp` to `StatusResponse`. build/vet/lint green.

- [x] **Validated end-to-end (2026-06-17)**: increase 0→1 in 3s, decrease observed at 13m16s.
- [x] **Re-validated (2026-07-11, target 3, dilute-to-min-validity + drop)**: 0→3 in ~64 MiB /
      ~6 min, pull-sync idled, diluted a batch 48h→24h→12h→6h (floored at the min-validity), radius
      3→2 at ~8 min, check passed. Needs the patched image (the default registry image is unpatched
      — the radius won't climb).

## Phase 2.5 — the decrease lever: pull-sync idle (local) / batch expiry (testnet) (done)

Two paths reach the decrease, both gated on `SyncRate()==0`:

- **Local (patched cluster)** — the reserve worker's gate (`reserve.go`:
  `countWithinRadius < threshold && SyncRate()==0 && radius > minimumRadius`) reduces to just
  **`SyncRate()==0`** (`threshold` patched to 100%, `countWithinRadius` well below it), so once
  pull-sync idles the radius ticks down on its own — measured ~8–13 min after idle. `waitPullSyncIdle`
  (`pullsyncRate<=0.05`; the rate settles to a small residual, never exactly 0) opens that gate.
- **Testnet (real reserve)** — the decrease is **commitment-driven**: it needs a batch to expire so
  its chunks are evicted. `force-decrease` **dilutes one batch to the chain's `minimumValidityBlocks`
  floor** (each `+1` ~halves TTL; the contract reverts any dilution below the floor, so expiry itself
  is unreachable — dilution only *hastens* it) so the batch expires in ~a few hours instead of its
  full TTL. Requires a non-zero chain price (see the `change-storage-price` skill); with price 0
  batches have infinite TTL. Full mechanics: [`docs/radius-halt.md`] and the radius-testing notes.

The general `load` primitives added along the way remain available but are not wired into the radius
check: the `--parallel` check flag (`check.go` + `runner.go`) and `load`'s re-arming `decrease-hold`.

Run: `beekeeper check --checks=ci-reserve-radius --timeout=45m` (local; raise `decrease-timeout`/`timeout` on testnet).

## Phase 3 — Real ephemeral cluster

- [ ] Run the check against an unpatched ~20-node ephemeral cluster (realistic timing).
- [ ] Wire as a periodic / public-testnet RC check (per #591), not standard PR CI.

## A/B testing procedure (current vs fixed bee)

Run the *identical* radius-decrease scenario against two builds and diff them:

- **A = current bee** (stock `master`) — expected to **reproduce** the halt.
- **B = fixed bee** (@sig's SWIP-25 PR) — expected to **survive** it.

## Open questions

- Patch vs config: can the small-reserve setup come from `bee-config` knobs alone?
- Does `/redistributionstate` need full-mode + incentives locally, or assert on `/status` +
  `/reservestate` only?
- Metrics sink for local runs: pushgateway vs scrape vs `radius-poll.sh` CSV.

## References

- Skill: `.claude/skills/radius-testing/SKILL.md` (+ `scripts/radius-poll.sh`)

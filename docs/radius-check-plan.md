# Reserve-radius change check — plan

Status: **draft / not started**. Owner: TBD. Last updated: 2026-06-16.

> **Working agreement**
> - Build the check on a **new feature branch off `master`** (e.g.
>   `feat/reserve-radius-check`), never directly on `master`. The patched-local
>   experiments and the bee-side patch may also need a coordinated bee branch.
> - The check must follow the **standardized behaviour of existing checks** — same
>   shape, helpers, and idioms as `smoke` / `feed` / `load` (random full-node
>   selection via `cluster.ShuffledFullNodeClients`, seeded RNG, `PostageTTL`-based
>   batches, `metrics.Reporter`). See "Standard check conventions" below; do not
>   invent a new pattern.

A repeatable, automated way to stage a **storage-radius change** across a Bee
cluster and capture what happens to the reserve and to pull-sync afterwards. The
end product is a beekeeper check (`pkg/check/reserveradius`) plus the metrics it
emits. Operating know-how lives in the `radius-testing` skill; this doc is the
design and the build checklist.

## Why this is hard (and what prior tryouts taught us)

- A stock node's reserve is far too large to overflow in CI-time, so a radius
  transition won't happen on a small local cluster without help.
- After a radius change, the node **resyncs its reserve over pull-sync**, and the
  redistribution-game "fully synced" status takes a long time to settle. So the
  test is **stage → monitor over time**, not a one-shot assertion.
- **PR #581** (`radiusdecrease`, open, unmerged) — deliberately forces a radius
  *decrease* (overflow a tiny reserve to push `storageRadius` 0→1, let it fall
  1→0) and polls `/status` for `pullsyncRate > 0` to confirm puller recovery.
  Reusable `waitForRadius` / `waitForRecovery` poll loops. **Requires a patched
  bee image** (`master-scenario-b`) + a coordinated bee PR; no review; carries
  dirty local config. The real radius tryout.
- **PR #591** (`stampexpiry`, draft) — patch-free, observes radius only as a side
  effect of stamp expiry. Review feedback (unactioned): use `PostageTTL`, pick a
  **random** node not `sortedNodes[0]`, account for the ~10-block batch-usable
  delay, and run as a **periodic / public-testnet RC** check, not standard CI.
- **GC check** (`pkg/check/gc/reserve.go`) — the precedent for a check that
  depends on a bee patch: patches live in `bee/.github/patches/`, applied by
  bee's beekeeper workflow before the check runs.

## Signals (see the `radius-testing` skill for the full table)

Per node, HTTP API (`http://<node>.localhost`):
`/status` → `storageRadius`, `pullsyncRate`, `reserveSize`,
`reserveSizeWithinRadius`, `committedDepth`; `/reservestate` → `radius`,
`commitment`; `/redistributionstate` (full-mode only) → `isFullySynced`, `phase`.
Prometheus: `bee_localstore_storage_radius`, `bee_localstore_reserve_size*`,
`bee_pullsync_chunks_*`, `bee_postage_radius`, `bee_storageincentives_*`.

## Approach: patched-local first, then real ephemeral

**Decision:** start with a **patched local cluster** for deterministic, fast radius
transitions; graduate to an **unpatched ephemeral k8s cluster** for realistic
timing (periodic/RC cadence). Both phases drive the same check; only the
environment and timeouts differ.

---

## Phase 0 — Local loop scaffolding (mostly done)

- [x] `/cluster-verify` command — verify substrate + Bee nodes + capture a
      reserve/radius baseline.
- [x] `/cluster-up` / `/cluster-down` commands — wrap beelocal + docker + beekeeper.
- [x] `radius-testing` skill + `scripts/radius-poll.sh` — the monitoring poller.
- [ ] Run `/cluster-up local-dns`, `/cluster-verify local-dns`, confirm a clean
      baseline (`storageRadius==0`, full nodes connected).

## Staging the change: driving the radius via uploads (reuse the `load` check)

You do **not** pre-compute "how many bytes" to upload — you upload in a loop, watch the
radius signal, and stop at a target. The `load` check already does exactly this:

- `load` (`pkg/check/load/load.go`) has `MaxCommittedDepth` and gates every upload on
  `checkCommittedDepth(client, max, wait)`, which reads `client.Status(ctx).CommittedDepth`
  and keeps uploading while `CommittedDepth < max` (and *waits* once the cap is reached).
  Since `committedDepth = storageRadius + capacityDoubling`, capping committedDepth caps radius.
  `ci-load` in `config/local.yaml` already wires `max-committed-depth: 2`.

**Increase and decrease are NOT symmetric — this is the crux:**

- **Increase** is upload-driven: fill the reserve past capacity ⇒ eviction ⇒ `storageRadius`
  rises. `load` already produces this.
- **Decrease** is *not* upload-driven. Per `bee/pkg/storer/reserve.go`, radius decreases only
  when the reserve is **under-utilized** AND the node is **fully synced** (`SyncRate()==0`) AND
  above `minimumRadius`. Trigger it by first inflating radius, then letting the patched
  (tiny-reserve, threshold=capacity, fast wake-up) reserve worker re-evaluate — PR #581's
  "overflow to push 0→1, then watch it fall 1→0" cascade. **The resync-over-decrease is the
  mechanism that is not yet understood, so it must be observed (spike) before it is designed.**

**Decisions:**
- The final `reserveradius` check **drives its own upload** (it needs tight
  stage→poll→record→assert control), **reusing load's committedDepth-gated upload primitive**
  — factor it into a shared helper rather than copy/paste. Smaller batches = finer radius control.
- For the **initial spike**, drive uploads with the existing `ci-load` check (set
  `max-committed-depth` to the target) while `radius-poll.sh` records the timeline — no new Go.

## Phase 1 — Empirical spike: force a change with a patched bee, observe decrease + resync

- [x] **Patch created** (in `bee/.github/patches/`, GC-check pattern). All three symbols
      are unreachable by config — `ReserveCapacity` = `(1<<doubling)*DefaultReserveCapacity`
      in `node.go:300` (no shrink flag); `ReserveWakeUpDuration` and `threshold()` aren't flags:
      - `radius_threshold.patch` → `pkg/storer/reserve.go:45` — `threshold` `capacity*5/10`(50%) → `capacity`(100%).
      - `radius_reserve.patch` → `pkg/storer/storer.go:251` `DefaultReserveCapacity` `1<<22`→`200`;
        `:411` `ReserveWakeUpDuration` `30m`→`10s` (drives `thresholdTicker` at reserve.go:130).
      gofmt-clean, verified to apply cleanly. `minimum-storage-radius` defaults to 0 so 1→0 is allowed.
- [ ] **Apply, build, push, redeploy** (revert source after build so the bee tree stays clean):

```sh
cd <bee-repo>
patch pkg/storer/reserve.go .github/patches/radius_threshold.patch
patch pkg/storer/storer.go  .github/patches/radius_reserve.patch
make docker-build PLATFORM=linux/arm64 BEE_IMAGE=k3d-registry.localhost:5000/ethersphere/bee:latest \
  REACHABILITY_OVERRIDE_PUBLIC=true BATCHFACTOR_OVERRIDE_PUBLIC=2
docker push k3d-registry.localhost:5000/ethersphere/bee:latest
git checkout pkg/storer/reserve.go pkg/storer/storer.go
```

Then redeploy so nodes pull the patched `:latest` (imagePullPolicy: Always):
`/cluster-down local-dns` → `/cluster-up local-dns` (or `kubectl rollout restart statefulset -n local`).
To wire into CI later, add the two `patch` lines to bee's `.github/workflows/beekeeper.yml`
"Apply patches and build" step (next to `postage_api`/`retrieval`).

- [ ] **Drive the increase** with the existing `ci-load` check (no new Go): set
      `max-committed-depth` to the target so it uploads until `storageRadius`/`committedDepth`
      reaches it. Run `radius-poll.sh` alongside to record the 0→target timeline.
- [ ] **Observe the decrease + resync** — stop uploads and watch whether `storageRadius`
      falls back and how pull-sync behaves (`pullsyncRate`, `reserveSize`,
      `reserveSizeWithinRadius` over time). Capture the full CSV timeline; use `/loop` for the
      long watch. This answers the open question of *how* resync-over-radius-decrease works.
- [x] **Gate met** — spike reproduced a real increase, decrease, and resync (findings below).

### Spike findings (2026-06-17, local-dns, patched `:200/10s/100%` image)

Sequence observed (3 full nodes, uploading 1 MB blobs to `bee-0`):

1. **Increase is immediate** — `storageRadius` 0→1 after ~2 uploads (~2 MB), 0→2 within ~14 s.
   On stock capacity it never moves; the patch is what makes it observable.
2. **Overshoot after uploads stop** — radius kept climbing (1→2) *after* uploads ceased,
   because in-flight **pushsync** keeps filling reserves. "Uploads stopped" ≠ "radius settled".
3. **Decrease lags ~10 min and is gated on stabilization** — the 2→1 decrease fired at
   ~10.5 min after node start (`node "Sync status check evaluated" stabilized=true`), ~30 s
   after a 10-min watch window ended. The reserve worker's decrease loop sits behind the
   startup-stabilizer gate (`reserve.go` `startReserveWorkers` waits on `startupStabilizer`),
   so a fresh node won't decrease for several minutes regardless of reserve state.
4. **Puller drives the reconfiguration** — `node/puller "radius decrease" old=2 new=1`, preceded
   by a storm of `syncWorker context cancelled` (disconnect/reconnect per bin). This is the
   PR #581 `manage()`/`disconnectPeer()` path — the liveness risk lives here.
5. **Decrease ↔ resync interlock (staircase)** — decrease requires `SyncRate()==0`, but a decrease
   triggers pull-sync (`pullsyncRate>0`), which then blocks the next decrease until it drains.
   Radius settles in steps, not one jump.

**Implications for the Phase 2 check (bake these in):**

- Use a **random full node** but expect the transition to show on whichever node's neighborhood
  fills — assert on the cluster, not one node.
- **Long, staged timeouts**: increase ~minutes; decrease/settle **≥15 min** (cf. PR #581's 20-min
  recovery). Don't fail fast.
- "Radius reached target" must wait for **pushsync to drain** (overshoot) before treating a value
  as settled — poll until `storageRadius` is stable for K ticks AND `pullsyncRate==0`, not a single read.
- Gate the decrease assertion on **`isWarmingUp==false`/stabilized** first; before that, no decrease can occur.
- The key liveness signal is `pullsyncRate>0` returning after the decrease (puller workers
  recovered) — the same assertion PR #581 makes. A stuck `manage()` shows as `pullsyncRate` flat at 0.
- `radius-poll.sh` with `-i 15` + auto-stop-on-stable is the right monitor; the in-repo check
  formalizes this loop.

## Standard check conventions (match existing checks — don't invent a pattern)

Copy the shape of `smoke` (`pkg/check/smoke/smoke.go`), `feed`, `load`. Concretely:

- **Boilerplate:** `Options` struct + `NewDefaultOptions() Options`; compile check
  `var _ beekeeper.Action = (*Check)(nil)`; `Check{ metrics, logger }`;
  `NewCheck(log logging.Logger) beekeeper.Action` → `&Check{metrics: newMetrics("check_reserve_radius"), logger: log}`.
- **Run signature:** `Run(ctx context.Context, cluster orchestration.Cluster, opts any) error`,
  first line `o, ok := opts.(Options); if !ok { return errors.New("invalid options type") }`.
- **Seeded RNG + random node selection** (the part the user called out — `smoke.go:116-131`).
  Take `[0]` of the shuffled list; never hard-index an unshuffled list / always-node-0 (the
  exact thing flagged in PR #591 review):

  ```go
  rnd := random.PseudoGenerator(o.RndSeed)              // pkg/random
  fullNodeClients, err := cluster.ShuffledFullNodeClients(ctx, rnd)  // (ctx, *rand.Rand) (ClientList, error)
  if err != nil { return fmt.Errorf("get shuffled full node clients: %w", err) }
  if len(fullNodeClients) < 1 { return fmt.Errorf("reserve-radius check requires at least 1 full node, got %d", len(fullNodeClients)) }
  node := fullNodeClients[0]                            // a random node, since the list is shuffled
  c.logger.Infof("random seed: %d", o.RndSeed)
  ```

- **Postage:** use `node.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)`
  with `PostageTTL time.Duration` (default `24h`), **not** a raw amount.
- **Options idioms:** `RndSeed int64` defaulting to `time.Now().UnixNano()`; durations as
  `time.Duration`; a `Duration` + `scheduler.NewDurationExecutor(...)` if it's a long/repeating run.
- **Loops:** `select { case <-ctx.Done(): return nil; default: }` guard each iteration.
- **Errors:** wrap with context (`fmt.Errorf("...: %w", err)`); failure messages name the
  suspected cause + the timeout hit.
- **Tests:** external `package reserveradius_test`; race detector clean.

## Phase 2 — The beekeeper check (`pkg/check/reserveradius`)

On a **new branch** (`feat/reserve-radius-check`). Mirror the existing check structure
(`Action` interface, `NewCheck(logger)`, `metrics.Reporter`, the conventions above);
register in `pkg/config/check.go` and add a `ci-reserve-radius` entry to `config/local.yaml`.

- [ ] `pkg/check/reserveradius/reserveradius.go`
      - `Options`: target direction (increase/decrease), upload size, postage
        (`PostageTTL` not raw amount), poll interval, overflow/recovery timeouts, seed.
      - `Run`: (1) pick a random full node via `cluster.ShuffledFullNodeClients`;
        (2) record baseline `/status` + `/reservestate`; (3) **stage** the change
        (buy batch, upload to overflow / drive decrease); (4) **monitor**: poll
        until `storageRadius` reaches target and then `pullsyncRate` returns to 0
        (reuse #581's `waitForRadius` / `waitForRecovery`); (5) assert + emit metrics.
      - Fail messages must name the suspected cause (e.g. puller `manage()` stuck,
        cf. #581) and the timeout hit.
- [ ] `pkg/check/reserveradius/metrics.go` — emit: `storage_radius` gauge per node
      over time, `time_to_radius_change_seconds`, `time_to_resync_seconds`
      (pullsync→0), `reserve_size` / `reserve_size_within_radius` gauges,
      `pullsync_rate` gauge. Implement `Report()`.
- [ ] Register in `pkg/config/check.go` `Checks` map + `NewOptions` decoder.
- [ ] `config/local.yaml`: `ci-reserve-radius` check + (if needed) a patched
      node-group/bee-config. Generous timeout.
- [ ] Gate: `make build && make vet && make lint && make test`.

## Phase 3 — Real ephemeral cluster, no patch

- [ ] Run the same check against an unpatched ephemeral k8s cluster with realistic
      reserve sizes and timing; expect long durations.
- [ ] Wire it as a **periodic / public-testnet RC** check (per #591 review), not
      standard PR CI. A/B candidate bee versions by comparing the emitted metrics.

## Open questions

- Increase, decrease, or both? Decrease exercises the known puller bug (#581) and
  is the higher-value regression target; increase is the common steady-state path.
- Patch vs config: how much of the small-reserve setup can come from `bee-config`
  knobs alone, avoiding a source patch entirely?
- Does `/redistributionstate` need full-mode + incentives enabled on the local
  cluster, or do we assert purely on `/status` + `/reservestate`?
- Metrics sink for local runs: pushgateway vs scrape vs the CSV from `radius-poll.sh`.

## References

- Skill: `.claude/skills/radius-testing/SKILL.md` (+ `scripts/radius-poll.sh`)
- Prior art: PR #581, PR #591, `pkg/check/gc/reserve.go`, `pkg/check/pingpong/`,
  `pkg/check/smoke/metrics.go`
- Bee internals: `bee/pkg/storer/reserve.go`, `bee/pkg/postage/batchstore/store.go`,
  `bee/pkg/api/{status,postage,redistribution}.go`

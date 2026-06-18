# reserve-radius check

Stages a **storage-radius change** across a Bee cluster and verifies the node
recovers afterwards. It drives the radius up by uploading, then watches for the
radius to come back down and for pull-sync to react — catching regressions in the
puller's reaction to a radius change (the `manage()` / `disconnectPeer()` path that
caused the liveness bug in beekeeper PR #581).

> Keep this file in sync with the code. If you change `Options`, the `Run` flow, the
> emitted metrics, the registration, or the bee-patch requirement, update the matching
> section below in the same change.

## What it does (`Run` flow)

`reserveradius.go`, in order:

1. **`waitForWarmupDone`** — block until every observed node reports
   `isWarmingUp == false`. The reserve worker's *decrease* loop is gated on
   stabilization, so this is a precondition for the radius ever dropping.
2. **baseline** — snapshot `/status` for all observed nodes.
3. **`driveIncrease`** — buy a mutable batch on a random full node, then upload
   `BlobSize` random blobs to it until any observed node's `storageRadius` reaches
   `TargetRadius` (or `MaxUploads` / `IncreaseTimeout`). Tracks a **per-node
   high-water `peak`**.
4. **settle** — keep polling for `SettleWait`, still raising `peak`. This matters:
   on an already-stabilized node the decrease can begin *during* settle, so `peak`
   must be the max seen, not a single post-settle read (otherwise it reads back at
   baseline and no decrease is ever detectable).
5. **`observeDecrease`** — poll until any node's `storageRadius` falls below its
   `peak` (success), watching `pullsyncRate` for recovery along the way. If no
   decrease occurs within `DecreaseTimeout`, fail with a message pointing at the
   PR #581 puller stall.

## Options

Defaults in `NewDefaultOptions()`; YAML keys (kebab-case) are wired in
`pkg/config/check.go` under the `reserve-radius` entry.

| field | yaml | default | purpose |
| --- | --- | --- | --- |
| `RndSeed` | `rnd-seed` | `time.Now().UnixNano()` | seed for `random.PseudoGenerator` → shuffled node pick |
| `PostageTTL` | `postage-ttl` | `24h` | batch TTL (use TTL, not a raw amount) |
| `PostageDepth` | `postage-depth` | `22` | batch depth |
| `PostageLabel` | `postage-label` | `reserve-radius` | batch label |
| `UploadGroups` | `upload-groups` | `[bee]` | node groups to upload to / observe (empty = all full nodes) |
| `BlobSize` | `blob-size` | `1048576` (1 MiB) | bytes per upload |
| `MaxUploads` | `max-uploads` | `60` | cap on uploads in the increase phase |
| `TargetRadius` | `target-radius` | `1` | storageRadius to reach before stopping uploads |
| `WarmupWait` | `warmup-wait` | `15m` | max wait for nodes to finish warmup |
| `IncreaseTimeout` | `increase-timeout` | `5m` | max time to reach `TargetRadius` |
| `SettleWait` | `settle-wait` | `1m` | post-upload window (pushsync drain + peak tracking) |
| `DecreaseTimeout` | `decrease-timeout` | `20m` | max time to observe a decrease |
| `PollInterval` | `poll-interval` | `15s` | poll cadence |

## Metrics (`metrics.go`)

Emitted via the `metrics.Reporter` interface (pushed to the pushgateway when the
check runs with `--metrics-enabled`). Namespace `beekeeper`, subsystem
`check_reserve_radius`:

- `…_storage_radius{node}` — gauge, storageRadius per node
- `…_reserve_size{node}` — gauge, reserve size (chunks) per node
- `…_pullsync_rate{node}` — gauge, `/status` pullsyncRate per node
- `…_time_to_increase_seconds` — gauge, first-upload → `TargetRadius`
- `…_time_to_decrease_seconds` — gauge, uploads-stopped → first observed decrease

## Requirements

- **Patched bee image.** On stock capacity the reserve is far too large to move.
  The check needs a node built with `bee/.github/patches/radius_reserve.patch`
  (`DefaultReserveCapacity`→200, `ReserveWakeUpDuration`→10s) and
  `radius_threshold.patch` (decrease `threshold`→100%). See
  `docs/radius-check-plan.md` and the `radius-testing` skill.
- **`isWarmingUp` in `StatusResponse`** (`pkg/bee/api/status.go`) — the node returns
  it; the struct field was added for this check's warmup gate.

## Running it

Against a patched local cluster (`local-dns`), via the `ci-reserve-radius` config
entry (timeout 45m):

```sh
./dist/beekeeper check --cluster-name=local-dns --checks=ci-reserve-radius --log-verbosity=info
# add --metrics-enabled=true --metrics-pusher-address=localhost:9091 to push metrics
```

## Observed behavior (local, patched cluster)

- **Increase is fast** — ~1 MiB moves `storageRadius` 0→1 in seconds.
- **Decrease lags ~13 min** and is stabilization-gated; sized by `DecreaseTimeout` (20m).
- **No fixed floor** — the equilibrium radius depends on data volume (a data-heavy
  node can stay elevated and never decrease in-window). The check asserts *a decrease
  occurred*, not that the radius returns to a specific value.
- **Pull-sync on a radius change is a puller reconfiguration**: `bee_puller_worker`
  contracts on increase / expands on decrease, with a `bee_puller_worker_errors`
  cancel-storm. Actual resync volume (`bee_pullsync_chunks_delivered`) is ~0 on a
  near-empty cluster — meaningful resync needs data (Phase 3 / real cluster).

## Known limitations / follow-ups

- Pull-sync recovery is **observed and logged, not asserted** (`observeDecrease` has a
  `TODO`). `/status` `pullsyncRate` is a poor signal on light clusters (stays ~0); a
  robust assertion would scrape node `/metrics` (`bee_puller_worker`,
  `bee_pullsync_chunks_delivered`) and check the worker set reconfigures and recovers.
- Direction is fixed to increase-then-decrease; `increase`-only / `both` not yet supported.
- No unit tests yet (external `_test` package TBD).

## Related

- `docs/radius-check-plan.md` — design, phases, spike findings.
- `.claude/skills/radius-testing/` — operating know-how, `radius-poll.sh`, the
  Pushgateway/Prometheus/Grafana metrics stack (`metrics/`).
- Prior art: PR #581 (`radiusdecrease`), PR #591 (`stampexpiry`), `pkg/check/gc`,
  `pkg/check/load` (the committedDepth-gated upload primitive), `pkg/check/smoke`.

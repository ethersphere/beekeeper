# reserve-radius check

Drives a **storage-radius change** on a Bee cluster and verifies the radius comes
back **down** once pull-sync settles — exercising the reserve worker's decrease path
(the puller reconfiguration that caused the liveness bug in beekeeper PR #581).

One self-driving run pushes the radius **up** by uploading, then stops and waits for
pull-sync to go idle — the reserve worker only decreases the radius while
`SyncRate()==0` (== `/status pullsyncRate`) — optionally dilutes a batch down to the chain's
minimum validity so it expires soon (hastening the commitment drop), and confirms the radius
ticks back down.

> Keep this file in sync with the code. If you change `Options`, the `Run` flow, the
> emitted metrics, the registration, or the bee-patch requirement, update the matching
> section below in the same change.

## What it does (`Run` flow)

A single flow — there are no modes.

1. **`selectNodes`** — shuffle the full-node clients (seeded by `rnd-seed`), optionally
   filtered to `upload-groups`.
2. **`waitForWarmupDone`** — block until every observed node reports `isWarmingUp == false`.
   The reserve worker's decrease loop is gated on stabilization, so this is a precondition
   for the radius ever dropping.
3. **baseline** — snapshot `/status` for all observed nodes.
4. **`driveIncrease`** — upload `BlobSize` random blobs to **randomly-chosen** nodes (each
   under its own mutable batch tagged with `postage-label`) until any node's `storageRadius`
   reaches `TargetRadius`, tracking a per-node high-water `peak`. A transient per-node batch
   error is logged and skipped (another random node is picked next).
5. **`waitPullSyncIdle`** — stop uploading and wait until every node reports
   `pullsyncRate <= 0.05` (the rate settles to a small residual, never exactly 0), or
   `SyncSettleWait` elapses (then it proceeds anyway). This opens the decrease gate.
6. **`diluteToMinValidity`** (when `force-decrease`) — dilute one batch, `+DiluteStep` at a
   time (each step ~halves its remaining TTL), until a dilution is rejected because it would
   drop validity below the chain's `minimumValidityBlocks` floor. Dilution **cannot reach
   expiry** (the floor stops it, ~a few hours of validity), so this only *hastens* the natural
   expiry — the batch now expires within the check's budget instead of its full TTL. On expiry
   its chunks are evicted and reserve commitment drops. Diluting a single batch is a partial
   expiry (drops commitment without evicting the whole reserve, which would halt).
7. **`observeDecrease`** — watch for any node's `storageRadius` to fall below its `peak`.
   The first decrease passes the check; no decrease within `DecreaseTimeout` fails it (a
   stuck puller keeps `SyncRate>0`, so the radius never decreases — cf. PR #581).

**Two decrease paths.** On the patched local cluster the reserve worker's idle tick drops the
radius ~10 min after pull-sync idles (before any expiry) — validated live: 0→3 in ~64 MiB, then
3→2 at ~8 min. On testnet (real reserve, a chain price set) the decrease is **expiry-driven**:
the diluted batch expires ~`minimumValidityBlocks` after the dilution (a few hours), evicting
its chunks; raise `decrease-timeout` (~6h) and `timeout` (~8h) accordingly.

## Options

Defaults in `NewDefaultOptions()`; YAML keys (kebab-case) are wired in
`pkg/config/check.go` under the `reserve-radius` entry.

| field | yaml | default | purpose |
| --- | --- | --- | --- |
| `RndSeed` | `rnd-seed` | `time.Now().UnixNano()` | seed for `random.PseudoGenerator` → shuffled node pick + random uploader choice |
| `PostageTTL` | `postage-ttl` | `48h` | batch TTL; must exceed the chain's 24h minimum validity once a price is set (24h lands exactly on the boundary and is rejected) |
| `PostageDepth` | `postage-depth` | `22` | batch depth |
| `PostageLabel` | `postage-label` | `reserve-radius` | this check's batch label |
| `UploadGroups` | `upload-groups` | `[bee]` | node groups to observe and upload to |
| `BlobSize` | `blob-size` | `8388608` (8 MiB) | bytes per upload (bigger blobs reach a higher target radius faster) |
| `MaxUploads` | `max-uploads` | `200` | cap on uploads in the increase phase |
| `TargetRadius` | `target-radius` | `3` | storageRadius any node must reach before stopping uploads (≥3 leaves headroom above the effective floor) |
| `ForceDecrease` | `force-decrease` | `true` | dilute one batch to minimum validity to hasten its expiry; `false` = idle-tick only |
| `DiluteStep` | `dilute-step` | `1` | depth increase per dilution (~halves TTL); larger jumps revert once they cross the min-validity floor |
| `MaxDilutions` | `max-dilutions` | `20` | cap on dilutions when driving a batch to the validity floor |
| `GasPrice` | `gas-price` | `""` | gas price for the dilute tx (empty = node default) |
| `WarmupWait` | `warmup-wait` | `15m` | max wait for nodes to finish warmup |
| `IncreaseTimeout` | `increase-timeout` | `30m` | max time to reach `TargetRadius` |
| `SyncSettleWait` | `sync-settle-wait` | `10m` | max wait for `pullsyncRate<=0.05` on all nodes (the decrease gate); proceeds anyway on timeout |
| `DecreaseTimeout` | `decrease-timeout` | `6h` | max time to observe a radius decrease (local idle tick ~10min; testnet expiry ~a few hours) |
| `PollInterval` | `poll-interval` | `15s` | poll cadence |

## Metrics (`metrics.go`)

Emitted via the `metrics.Reporter` interface (pushed to the pushgateway when the
check runs with `--metrics-enabled`). Namespace `beekeeper`, subsystem
`check_reserve_radius`:

- `…_storage_radius{node}` — gauge, storageRadius per node
- `…_reserve_size{node}` — gauge, reserve size (chunks) per node
- `…_reserve_within_radius{node}` — gauge, reserve chunks within the storage radius (completeness signal)
- `…_pullsync_rate{node}` — gauge, `/status` pullsyncRate per node
- `…_time_to_increase_seconds` — gauge, first upload → `TargetRadius`
- `…_time_to_decrease_seconds` — gauge, pull-sync idle → first observed radius decrease
- `…_dilution_total` — counter, batch dilutions applied when hastening expiry (force-decrease)

## Requirements

- **Patched bee image.** On stock capacity the reserve is far too large to move.
  The check needs a node built with `bee/.github/patches/radius_reserve.patch`
  (`DefaultReserveCapacity`→4000, `ReserveWakeUpDuration`→10s) and
  `radius_threshold.patch` (decrease `threshold`→100%). See
  `docs/radius-check-plan.md` and the `radius-testing` skill. The default local-cluster
  image is **not** patched — build and push it first, or the radius never climbs.
- **A non-zero postage price is optional.** The decrease is driven by pull-sync idling
  (`SyncRate()==0`), not by batch expiry, so it works at price 0 (batches never expire).
  If a price *is* set (see the `change-storage-price` skill), the contract enforces a 24h
  minimum batch validity, so `postage-ttl` must exceed 24h — hence the `48h` default.
- **`isWarmingUp` in `StatusResponse`** (`pkg/bee/api/status.go`) — the node returns it;
  the struct field was added for this check's warmup gate.

## Running it

Against a patched local cluster (`local-dns`):

```sh
./dist/beekeeper check --cluster-name=local-dns --checks=ci-reserve-radius \
  --timeout=45m --metrics-enabled=true --metrics-pusher-address=localhost:9091 --log-verbosity=info
```

## Observed behavior (local, patched cluster)

- **Increase**: ~48–64 MiB moves the max `storageRadius` 0→3 in ~4–6 min (random uploads
  distribute by proximity, so all nodes climb).
- **Local decrease is pull-sync-idle-gated**: once uploads stop and `pullsyncRate` settles, the
  reserve worker ticks the radius down (3→2) after ~8–13 min; it can then oscillate 2↔3. There
  is an effective floor (~2), so drive to `target-radius` ≥3 to leave headroom.
- **Dilution cannot reach expiry**: the postage contract reverts any dilution that would drop a
  batch below `minimumValidityBlocks` (~a few hours here), so a batch floors at ~6h validity and
  can't be diluted to 0. `force-decrease` therefore *hastens* the natural expiry (dilutes to the
  floor) rather than triggering it directly. Locally this is redundant with the idle tick (which
  fires first); it matters on testnet, where the decrease is expiry-driven. See
  `docs/radius-check-plan.md`.

## Related

- `docs/radius-check-plan.md` — design, phases, spike findings.
- `.claude/skills/radius-testing/` — operating know-how, `radius-poll.sh`, the metrics stack.
- `.claude/skills/change-storage-price/` — activating a non-zero postage price on the local chain.
- Prior art: PR #581 (`radiusdecrease`), PR #591 (`stampexpiry`), `pkg/check/gc`,
  `pkg/check/load`, `pkg/check/smoke`.

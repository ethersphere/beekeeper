# reserve-radius-v2 check

A **simplified, reproducible** reserve-radius scenario built to capture pull-sync issues
around storage-radius changes and postage-batch dilution.

Differences from v1 (`reserve-radius`):

- **Granular fill control** — instead of one large depth-22 batch per node, it creates
  **many small batches** (default depth 18) and uploads a **fixed amount of data per batch**
  (default 4 MiB ≈ 1024 chunks), so the reserve fill is known and reproducible.
- **Capacity-threshold targeting** — it fills until any node's `reserveSizeWithinRadius`
  is **just over 50%** of the configured reserve capacity (default target 55% of 4000
  chunks), the trigger point for the radius decrease.
- **Dilutions as distinct events** — after pull-sync settles, batches are diluted **one at
  a time** (default one per minute), each incrementing a plain `dilutions_total` counter so
  a Grafana time series shows exactly when dilutions occur relative to radius changes.

## Flow

1. Wait for all observed nodes to finish warmup.
2. **Fill**: create batches round-robin across nodes (label `<postage-label>-<n>`), upload
   `data-per-batch` bytes under each, stop when max fill ≥ `target-fill-percent`.
3. Wait for pull-sync to go idle (`pullsyncRate <= 0.05`, best-effort).
4. **Decrease**: dilute one batch every `dilute-interval` while polling; pass on the first
   `storageRadius` drop below its peak, fail after `decrease-timeout` (stuck pull-sync).

## Options

Defaults in `NewDefaultOptions()`; YAML keys wired in `pkg/config/check.go` under
`reserve-radius-v2`. Key ones:

| yaml | default | purpose |
| --- | --- | --- |
| `postage-amount` | `1000` | amount per batch (each batch is created fresh via `CreatePostageBatch`, never reused) |
| `batch-depth` | `18` | depth of each small batch |
| `blob-size` | `1048576` (1 MiB) | bytes per upload |
| `data-per-batch` | `4194304` (4 MiB) | bytes uploaded under each batch |
| `max-batches` | `50` | cap on batches in the fill phase |
| `reserve-capacity` | `4000` | reserve capacity in chunks (must match the bee patch) |
| `target-fill-percent` | `55` | stop filling when any node crosses this % of capacity |
| `dilute` | `true` | dilute batches to drive the decrease |
| `dilute-interval` | `1m` | spacing between dilutions (distinct chart events) |
| `max-dilution-rounds` | `5` | each round dilutes every batch once |
| `decrease-timeout` | `30m` | max wait for a radius decrease (raise to ~6h on testnet) |

## Metrics

Namespace `beekeeper`, subsystem `check_reserve_radius_v2`:

- `…_storage_radius{node}`, `…_reserve_size{node}`, `…_reserve_within_radius{node}`,
  `…_reserve_fill_percent{node}`, `…_pullsync_rate{node}` — gauges polled every `poll-interval`
- `…_radius_events_total{node,direction}` — counter of observed radius increases/decreases
- `…_batches_created_total`, `…_uploaded_bytes_total`, `…_dilutions_total` — counters
- `…_time_to_decrease_seconds` — gauge, decrease-phase start → first observed decrease

## Requirements

Same as v1: a **patched bee image** (`radius_reserve.patch`: capacity 4000, fast wakeup;
`radius_threshold.patch`) — on stock capacity the radius never moves. `reserve-capacity`
must match the patched value. A non-zero chain postage price is required for dilution to
have an effect (at price 0 batches have infinite TTL and dilution is skipped).

## Running it

```sh
./dist/beekeeper check --cluster-name=local-dns --checks=ci-reserve-radius-v2 \
  --timeout=90m --metrics-enabled=true --metrics-pusher-address=localhost:9091 --log-verbosity=info
```

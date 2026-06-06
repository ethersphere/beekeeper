# Smoke check refactor — design

Date: 2026-06-06
Scope: `pkg/check/smoke/` (`smoke.go`, `metrics.go`, `test.go`)

## Goal

Two outcomes, in one pass:

1. **Readability.** `c.run` is ~215 lines and five levels of nesting. Decompose it into
   focused, screen-sized functions so the upload → download flow reads top-down.
2. **Observable attempt → success → fail flow.** Today attempt, success, and failure are
   separate counter metrics, so on a Grafana chart they are independent lines that each only
   move on their own events. During a failure streak the success line flatlines and attempts
   drift away from both, so the flow can't be visualized on a single panel. Fix this by making
   the outcome a **label** on one counter per phase (the canonical Prometheus pattern).

## Non-goals

- No change to the node-selection behavior: the uploader/downloader pair is chosen once
  (`ShuffledFullNodeClients` → `clients[0]`/`clients[1]`) and stays fixed for the whole run.
  This is intentional and confirmed.
- No change to retry timing. The pre-attempt sleep (sleeping `TxOnErrWait`/`RxOnErrWait`
  before *every* attempt, including the first) is deliberate — it gives the cluster time to
  sync data across nodes — and is preserved exactly.
- No touching the vestigial option fields (`ContentSize`, and `UploadTimeout` /
  `DownloadTimeout` / `IterationWait`, which are not wired into the YAML decode in
  `pkg/config/check.go`). Out of scope.
- No new labels for `run_id` / `namespace`. The Grafana grouping problem was solved
  separately via query/panel changes; adding `run_id` to push-gateway metrics would be a
  high-cardinality anti-pattern (the gateway retains series indefinitely).

## Part A — Metrics reshape (`metrics.go`)

Hard-replace the nine attempt/success/error/mismatch counters with three result-labeled
counters. The `result` label is what lets a single panel show the whole flow.

| Today | Becomes |
|---|---|
| `BatchCreateAttempts`, `BatchCreateErrors` | `smoke_batch_total{result="success\|error"}` |
| `UploadAttempts`, `UploadErrors`, `UploadSuccess` | `smoke_upload_total{size_bytes, node_name, redundancy_level, result="success\|failure"}` |
| `DownloadAttempts`, `DownloadErrors`, `DownloadSuccess`, `DownloadMismatch` | `smoke_download_total{size_bytes, node_name, redundancy_level, result="success\|error\|mismatch"}` |

Invariant: **exactly one increment per attempt**, tagged with its outcome. Therefore
`attempts == sum by () over result` always — attempts and outcomes can never drift apart.
The existing retry loops already produce exactly one terminal outcome per try, so this is a
faithful 1:1 mapping, not a behavior change:

- Upload try → `success` on upload, else `failure`.
- Download try → `error` (transport/download failed), `mismatch` (downloaded but bytes
  differ), or `success` (downloaded and bytes match).
- Batch create → `success` or `error`.

The new metric name is the canonical name; full Prometheus name is
`beekeeper_check_smoke_{batch,upload,download}_total` (namespace `beekeeper`, subsystem
`check_smoke`).

**Kept unchanged** (not part of the attempt/fail flow, working fine, recorded only on
success): `UploadDuration` / `DownloadDuration` histograms, `UploadThroughput` /
`DownloadThroughput` gauges, `UploadedBytes` / `DownloadedBytes` counters.

`result` label constant added alongside the existing `labelSizeBytes` / `labelNodeName` /
`labelRedundancyLevel`.

### Resulting Grafana panel

```promql
sum by (result) (rate(beekeeper_check_smoke_upload_total[$__rate_interval]))
```

Stacked: total height = attempts, bands = success vs failure, shifting live during a failure
streak. Same shape for `download_total` (three bands) and `batch_total`.

## Part B — Code restructure (`smoke.go`)

Decompose `c.run` into focused functions. No behavior change — same order, same sleeps, same
retry counts, same logging.

- **`run(ctx, cluster, o)`** — setup (log config, seed `random.PseudoGenerator`, fetch
  `ShuffledFullNodeClients`, validate ≥2, pick `uploader`/`downloader` once, build `test.Test`),
  then the iteration loop: `GetOrCreateMutableBatch` (record `smoke_batch_total`; on error
  sleep `TxOnErrWait` + continue), iterate `resolveRLevels(o.RLevels) × o.FileSizes` calling
  `roundTrip`, then sleep `IterationWait`. `ctx.Done()` exits as today.
- **`roundTrip(ctx, t, uploader, downloader, batchID, size, rLevel, o)`** — make random data
  (`crypto/rand`), `upload`; on success sleep `NodesSyncWait` then `download`. On upload
  failure, skip download (as today).
- **`upload(...) (swarm.Address, bool)`** — owns the 3-try retry loop (sleep-before-each
  preserved), records `smoke_upload_total{result}` per try, and on success records
  duration/throughput/uploaded-bytes. Returns the address and whether it succeeded.
- **`download(...)`** — owns the 3-try retry loop (sleep-before-each preserved), classifies
  each try as `success` / `error` / `mismatch`, records `smoke_download_total{result}`, logs
  the byte-diff detail on mismatch, and logs downloader topology when all tries fail.
- **Pure helpers** (unit-tested — the currently empty `test.go` gets used):
  - `resolveRLevels([]*redundancy.Level) []*redundancy.Level` — empty → `[]{nil}`; replaces
    the manual index/`break` loop with a plain `range`.
  - `redundancyLevelLabel(*redundancy.Level) string` — unchanged (`nil` → `"not_set"`).
  - `countByteDiff(a, b []byte) int` — extract the mismatch byte-diff counting for clarity
    and testability.

## Testing

- Unit tests (external `package smoke_test`) for the pure helpers: `resolveRLevels`
  (empty and non-empty), `redundancyLevelLabel` (nil and set), `countByteDiff` (equal,
  length-mismatch, partial-diff).
- The orchestrated `run` loop is left to existing integration usage (needs a live/mocked
  cluster); the refactor preserves its behavior, verified by reading the diff.

## Verification checklist

- `make build`, `make vet`, `make lint`, `make test` pass.
- Diff confirms: same control flow, same sleeps, same retry counts, same log lines; only the
  metric recording calls change name/label.
- No references to the old metric field names remain in the package.

# Smoke Check Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `pkg/check/smoke` readable (decompose the 215-line `c.run`) and make the attempt→success→fail flow visualizable on a single Grafana panel by collapsing the per-outcome counters into one `result`-labeled counter per phase.

**Architecture:** Hard-replace the nine attempt/success/error/mismatch counters with three `*_total{result=...}` counters (one increment per attempt). Split `c.run` into `run`/`roundTrip`/`upload`/`download` plus pure helpers, passing the uploader/downloader tester through a small in-package `transferrer` interface (the concrete `test` type is unexported). Behavior — node selection, sleeps (including the pre-first-attempt sync sleep), retry counts, and log lines — is preserved exactly.

**Tech Stack:** Go 1.26, Cobra/Viper CLI, `prometheus/client_golang`, `github.com/ethersphere/bee/v2`.

**Spec:** `docs/superpowers/specs/2026-06-06-smoke-check-refactor-design.md`

---

## File Structure

- `pkg/check/smoke/metrics.go` — **rewrite**: metric struct + `newMetrics`; three `result`-labeled counters, the rest unchanged.
- `pkg/check/smoke/smoke.go` — **rewrite**: decomposed `run`/`roundTrip`/`upload`/`download`, `transferrer` interface, result constants, and pure helpers (`resolveRLevels`, `redundancyLevelLabel`, `countByteDiff`).
- `pkg/check/smoke/smoke_test.go` — **create**: internal (`package smoke`) unit tests for the pure helpers (they are unexported, so the test must be in-package).
- `pkg/check/smoke/test.go` — left untouched (stray empty `package smoke` file; out of scope).

Note: metrics.go and smoke.go must change together (smoke.go references the metric fields), so the reshape is one atomic task (Task 2). Task 1 only *adds* the two new pure helpers + their tests so they can be TDD'd in isolation; Task 2's full-file rewrite keeps them.

---

## Task 1: Pure helpers + unit tests (TDD)

**Files:**
- Modify: `pkg/check/smoke/smoke.go` (append two helper funcs)
- Create: `pkg/check/smoke/smoke_test.go`

- [ ] **Step 1: Write the failing tests**

Create `pkg/check/smoke/smoke_test.go`:

```go
package smoke

import (
	"testing"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
)

func TestResolveRLevels(t *testing.T) {
	t.Run("empty defaults to single nil level", func(t *testing.T) {
		got := resolveRLevels(nil)
		if len(got) != 1 {
			t.Fatalf("expected 1 level, got %d", len(got))
		}
		if got[0] != nil {
			t.Fatalf("expected nil level, got %v", got[0])
		}
	})

	t.Run("returns configured levels unchanged", func(t *testing.T) {
		l := redundancy.Level(1)
		in := []*redundancy.Level{&l}
		got := resolveRLevels(in)
		if len(got) != 1 || got[0] != &l {
			t.Fatalf("expected configured levels returned unchanged, got %v", got)
		}
	})
}

func TestRedundancyLevelLabel(t *testing.T) {
	if got := redundancyLevelLabel(nil); got != "not_set" {
		t.Fatalf("nil: expected not_set, got %q", got)
	}
	l := redundancy.Level(2)
	if got := redundancyLevelLabel(&l); got != "2" {
		t.Fatalf("level 2: expected \"2\", got %q", got)
	}
}

func TestCountByteDiff(t *testing.T) {
	tests := []struct {
		name string
		a, b []byte
		want int
	}{
		{"equal", []byte{1, 2, 3}, []byte{1, 2, 3}, 0},
		{"all differ", []byte{1, 2, 3}, []byte{4, 5, 6}, 3},
		{"some differ", []byte{1, 2, 3}, []byte{1, 9, 3}, 1},
		{"shorter b compares min length", []byte{1, 2, 3}, []byte{1, 2}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := countByteDiff(tc.a, tc.b); got != tc.want {
				t.Fatalf("countByteDiff(%v,%v)=%d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail to compile**

Run: `go test ./pkg/check/smoke/...`
Expected: build failure — `undefined: resolveRLevels`, `undefined: countByteDiff`. (`redundancyLevelLabel` already exists.)

- [ ] **Step 3: Add the two helper functions**

In `pkg/check/smoke/smoke.go`, append after the existing `redundancyLevelLabel` function:

```go
// resolveRLevels returns the configured redundancy levels, defaulting to a single
// nil level (redundancy disabled) when none are configured.
func resolveRLevels(levels []*redundancy.Level) []*redundancy.Level {
	if len(levels) == 0 {
		return []*redundancy.Level{nil}
	}
	return levels
}

// countByteDiff returns the number of differing bytes between a and b, comparing
// up to the length of the shorter slice.
func countByteDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	diff := 0
	for i := range n {
		if a[i] != b[i] {
			diff++
		}
	}
	return diff
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./pkg/check/smoke/...`
Expected: PASS (`ok  github.com/ethersphere/beekeeper/pkg/check/smoke`). The two new funcs are not yet used by production code — that is fine for `go build`/`go test` (unused *package-level* funcs are legal); they are wired up in Task 2 before lint runs in Task 3.

- [ ] **Step 5: Commit**

```bash
git add pkg/check/smoke/smoke.go pkg/check/smoke/smoke_test.go
git commit -m "test(smoke): add pure helpers resolveRLevels and countByteDiff with tests"
```

---

## Task 2: Reshape metrics + decompose run (atomic refactor)

This rewrites both files in one commit because smoke.go references the metric fields. No behavior change other than the metric names/labels and moving the (constant) uploader/downloader name logs out of the per-iteration loop.

**Files:**
- Modify: `pkg/check/smoke/metrics.go` (full rewrite)
- Modify: `pkg/check/smoke/smoke.go` (full rewrite)

- [ ] **Step 1: Rewrite `metrics.go`**

Replace the entire contents of `pkg/check/smoke/metrics.go` with:

```go
package smoke

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	BatchCreate        *prometheus.CounterVec
	Upload             *prometheus.CounterVec
	Download           *prometheus.CounterVec
	UploadDuration     *prometheus.HistogramVec
	DownloadDuration   *prometheus.HistogramVec
	UploadThroughput   *prometheus.GaugeVec
	DownloadThroughput *prometheus.GaugeVec
	UploadedBytes      *prometheus.CounterVec
	DownloadedBytes    *prometheus.CounterVec
}

const (
	labelSizeBytes       = "size_bytes"
	labelNodeName        = "node_name"
	labelRedundancyLevel = "redundancy_level"
	labelResult          = "result"
)

func newMetrics(subsystem string) metrics {
	return metrics{
		BatchCreate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "batch_total",
				Help:      "Number of batch create attempts by result.",
			},
			[]string{labelResult},
		),
		Upload: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_total",
				Help:      "Number of upload attempts by result.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel, labelResult},
		),
		Download: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_total",
				Help:      "Number of download attempts by result.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel, labelResult},
		),
		UploadDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_upload_duration",
				Help:      "Data upload duration through the /bytes endpoint.",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 600, 1200, 1800, 3600},
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		DownloadDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_download_duration",
				Help:      "Data download duration through the /bytes endpoint.",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 600, 1200, 1800, 3600},
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		UploadThroughput: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_throughput_bytes_per_second",
				Help:      "Upload throughput in bytes per second.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		DownloadThroughput: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_throughput_bytes_per_second",
				Help:      "Download throughput in bytes per second.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		UploadedBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "uploaded_bytes_total",
				Help:      "Total bytes successfully uploaded.",
			},
			[]string{labelNodeName, labelRedundancyLevel},
		),
		DownloadedBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "downloaded_bytes_total",
				Help:      "Total bytes successfully downloaded.",
			},
			[]string{labelNodeName, labelRedundancyLevel},
		),
	}
}

func (metrics *metrics) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(*metrics)
}
```

- [ ] **Step 2: Rewrite `smoke.go`**

Replace the entire contents of `pkg/check/smoke/smoke.go` with:

```go
package smoke

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/swarm"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/scheduler"
	"github.com/ethersphere/beekeeper/pkg/test"
	"github.com/prometheus/client_golang/prometheus"
)

// result label values for the smoke metrics.
const (
	resultSuccess  = "success"
	resultFailure  = "failure"
	resultError    = "error"
	resultMismatch = "mismatch"
)

// Options represents smoke test options
type Options struct {
	ContentSize     int64
	FileSizes       []int64
	RndSeed         int64
	PostageTTL      time.Duration
	PostageDepth    uint64
	PostageLabel    string
	TxOnErrWait     time.Duration
	RxOnErrWait     time.Duration
	NodesSyncWait   time.Duration
	Duration        time.Duration
	UploadTimeout   time.Duration
	DownloadTimeout time.Duration
	IterationWait   time.Duration
	RLevels         []*redundancy.Level
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ContentSize:     5000000,
		FileSizes:       []int64{5000000},
		RndSeed:         time.Now().UnixNano(),
		PostageTTL:      24 * time.Hour,
		PostageDepth:    24,
		PostageLabel:    "test-label",
		TxOnErrWait:     10 * time.Second,
		RxOnErrWait:     10 * time.Second,
		NodesSyncWait:   time.Minute,
		Duration:        12 * time.Hour,
		UploadTimeout:   60 * time.Minute,
		DownloadTimeout: 60 * time.Minute,
		IterationWait:   5 * time.Minute,
		RLevels:         []*redundancy.Level{},
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns new check
func NewCheck(log logging.Logger) beekeeper.Action {
	return &Check{
		metrics: newMetrics("check_smoke"),
		logger:  log,
	}
}

// transferrer uploads and downloads data against a bee node. It is satisfied by
// the concrete tester returned by test.NewTest (whose type is unexported).
type transferrer interface {
	Upload(ctx context.Context, c *bee.Client, data []byte, batchID string, rLevel *redundancy.Level) (swarm.Address, time.Duration, error)
	Download(ctx context.Context, c *bee.Client, addr swarm.Address, rLevel *redundancy.Level) ([]byte, time.Duration, error)
}

// Run creates a file of the specified size that is uploaded and downloaded.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return errors.New("invalid options type")
	}

	return scheduler.NewDurationExecutor(o.Duration, c.logger).Run(ctx, func(ctx context.Context) error {
		return c.run(ctx, cluster, o)
	})
}

func (c *Check) run(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	c.logger.Infof("random seed: %d", o.RndSeed)
	c.logger.Infof("testing file sizes: %v", o.FileSizes)
	c.logger.Infof("upload timeout: %s", o.UploadTimeout.String())
	c.logger.Infof("download timeout: %s", o.DownloadTimeout.String())
	c.logger.Infof("total duration: %s", o.Duration.String())

	rnd := random.PseudoGenerator(o.RndSeed)

	// Get shuffled full node clients for better load distribution and testing.
	fullNodeClients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("get shuffled full node clients: %w", err)
	}
	if len(fullNodeClients) < 2 {
		return fmt.Errorf("smoke check requires at least 2 full nodes, got %d", len(fullNodeClients))
	}

	// The uploader/downloader pair is fixed for the whole run by design.
	uploader := fullNodeClients[0]
	downloader := fullNodeClients[1]
	c.logger.Infof("uploader: %s", uploader.Name())
	c.logger.Infof("downloader: %s", downloader.Name())

	var t transferrer = test.NewTest(c.logger)

	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			c.logger.Infof("starting iteration: #%d", i)
		}

		batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
		if err != nil {
			c.metrics.BatchCreate.WithLabelValues(resultError).Inc()
			c.logger.Errorf("create new batch failed: %v", err)
			c.logger.Infof("retrying in: %v", o.TxOnErrWait)
			time.Sleep(o.TxOnErrWait)
			continue
		}
		c.metrics.BatchCreate.WithLabelValues(resultSuccess).Inc()
		c.logger.WithField("batch_id", batchID).Infof("node %s: using batch", uploader.Name())

		for _, rLevel := range resolveRLevels(o.RLevels) {
			for _, size := range o.FileSizes {
				select {
				case <-ctx.Done():
					return nil
				default:
				}
				c.roundTrip(ctx, t, uploader, downloader, batchID, size, rLevel, o)
			}
		}

		time.Sleep(o.IterationWait)
	}
}

// roundTrip uploads freshly generated random content of the given size, waits for
// the cluster to sync, then downloads and verifies it.
func (c *Check) roundTrip(ctx context.Context, t transferrer, uploader, downloader *bee.Client, batchID string, size int64, rLevel *redundancy.Level, o Options) {
	if rLevel != nil {
		c.logger.Infof("testing file size: %d bytes (%.2f KB), redundancy level: %d", size, float64(size)/1024, *rLevel)
	} else {
		c.logger.Infof("testing file size: %d bytes (%.2f KB), redundancy level: (not set)", size, float64(size)/1024)
	}

	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		c.logger.Errorf("unable to create random content for size %d: %v", size, err)
		return
	}

	address, ok := c.upload(ctx, t, uploader, batchID, data, rLevel, o)
	if !ok {
		return
	}

	time.Sleep(o.NodesSyncWait)

	c.download(ctx, t, downloader, address, data, rLevel, o)

	c.logger.Infof("completed testing file size: %d bytes", size)
}

// upload uploads data to the uploader, retrying up to three times. It sleeps
// TxOnErrWait before every attempt (including the first, to let the cluster sync).
// It records the per-attempt result and, on success, the duration/throughput/
// uploaded-bytes metrics. It returns the uploaded address and whether it succeeded.
func (c *Check) upload(ctx context.Context, t transferrer, uploader *bee.Client, batchID string, data []byte, rLevel *redundancy.Level, o Options) (swarm.Address, bool) {
	sizeLabel := strconv.Itoa(len(data))
	rLevelLabel := redundancyLevelLabel(rLevel)

	var (
		txCtx    context.Context
		txCancel context.CancelFunc = func() {}
	)
	defer func() { txCancel() }()

	for range 3 {
		txCancel()

		select {
		case <-ctx.Done():
			return swarm.ZeroAddress, false
		case <-time.After(o.TxOnErrWait):
		}

		txCtx, txCancel = context.WithTimeout(ctx, o.UploadTimeout)

		address, txDuration, err := t.Upload(txCtx, uploader, data, batchID, rLevel)
		if err != nil {
			c.metrics.Upload.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel, resultFailure).Inc()
			c.logger.Errorf("upload failed for size %d: %v", len(data), err)
			c.logger.Infof("retrying in: %v", o.TxOnErrWait)
			continue
		}

		c.metrics.Upload.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel, resultSuccess).Inc()
		c.metrics.UploadDuration.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel).Observe(txDuration.Seconds())
		c.metrics.UploadedBytes.WithLabelValues(uploader.Name(), rLevelLabel).Add(float64(len(data)))
		if txDuration.Seconds() > 0 {
			c.metrics.UploadThroughput.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel).Set(float64(len(data)) / txDuration.Seconds())
		}
		return address, true
	}

	c.logger.Infof("skipping download for size %d due to upload failure", len(data))
	return swarm.ZeroAddress, false
}

// download downloads addr from the downloader and verifies it matches want,
// retrying up to three times. It sleeps RxOnErrWait before every attempt. It
// records the per-attempt result (success/error/mismatch) and, on success, the
// duration/throughput/downloaded-bytes metrics. When every attempt fails it logs
// the downloader topology to aid debugging.
func (c *Check) download(ctx context.Context, t transferrer, downloader *bee.Client, addr swarm.Address, want []byte, rLevel *redundancy.Level, o Options) {
	sizeLabel := strconv.Itoa(len(want))
	rLevelLabel := redundancyLevelLabel(rLevel)

	var (
		rxCtx    context.Context
		rxCancel context.CancelFunc = func() {}
	)
	defer func() { rxCancel() }()

	for range 3 {
		rxCancel()

		select {
		case <-ctx.Done():
			return
		case <-time.After(o.RxOnErrWait):
		}

		rxCtx, rxCancel = context.WithTimeout(ctx, o.DownloadTimeout)

		data, rxDuration, err := t.Download(rxCtx, downloader, addr, rLevel)
		if err != nil {
			c.metrics.Download.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel, resultError).Inc()
			c.logger.Errorf("download failed for size %d: %v", len(want), err)
			c.logger.Infof("retrying in: %v", o.RxOnErrWait)
			continue
		}

		if bytes.Equal(data, want) {
			c.metrics.Download.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel, resultSuccess).Inc()
			c.metrics.DownloadDuration.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Observe(rxDuration.Seconds())
			c.metrics.DownloadedBytes.WithLabelValues(downloader.Name(), rLevelLabel).Add(float64(len(want)))
			if rxDuration.Seconds() > 0 {
				c.metrics.DownloadThroughput.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Set(float64(len(want)) / rxDuration.Seconds())
			}
			return
		}

		c.metrics.Download.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel, resultMismatch).Inc()
		c.logger.Infof("data mismatch for size %d: uploaded and downloaded data differ", len(want))

		if len(data) != len(want) {
			c.logger.Errorf("length mismatch for size %d: downloaded %d bytes, uploaded %d bytes", len(want), len(data), len(want))
			continue
		}

		diff := countByteDiff(want, data)
		c.logger.Infof("data mismatch for size %d: found %d different bytes, ~%.2f%%", len(want), diff, float64(diff)/float64(len(want))*100)
	}

	c.logger.Errorf("all download attempts failed for size %d, fetching downloader topology", len(want))
	top, err := downloader.Topology(ctx)
	if err != nil {
		c.logger.Errorf("failed to get downloader topology: %v", err)
		return
	}
	c.logger.Infof("downloader %s topology: depth=%d, connected=%d, population=%d, reachability=%s, bins=%s",
		downloader.Name(), top.Depth, top.Connected, top.Population, top.Reachability, top.Bins.String())
}

func (c *Check) Report() []prometheus.Collector {
	return c.metrics.Report()
}

// resolveRLevels returns the configured redundancy levels, defaulting to a single
// nil level (redundancy disabled) when none are configured.
func resolveRLevels(levels []*redundancy.Level) []*redundancy.Level {
	if len(levels) == 0 {
		return []*redundancy.Level{nil}
	}
	return levels
}

func redundancyLevelLabel(rLevel *redundancy.Level) string {
	if rLevel == nil {
		return "not_set"
	}
	return strconv.Itoa(int(*rLevel))
}

// countByteDiff returns the number of differing bytes between a and b, comparing
// up to the length of the shorter slice.
func countByteDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	diff := 0
	for i := range n {
		if a[i] != b[i] {
			diff++
		}
	}
	return diff
}
```

- [ ] **Step 3: Build and test**

Run: `make build && go test ./pkg/check/smoke/...`
Expected: build succeeds; smoke unit tests PASS. If the build fails on the `transferrer` interface, confirm `test.NewTest` returns a value whose `Upload`/`Download` method signatures match exactly (they do as of this writing — `pkg/test/test.go`).

- [ ] **Step 4: Commit**

```bash
git add pkg/check/smoke/metrics.go pkg/check/smoke/smoke.go
git commit -m "refactor(smoke): consolidate outcome counters and decompose run loop"
```

---

## Task 3: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Confirm no old metric field names remain**

Run: `grep -rn "BatchCreateAttempts\|BatchCreateErrors\|UploadAttempts\|UploadErrors\|UploadSuccess\|DownloadAttempts\|DownloadErrors\|DownloadSuccess\|DownloadMismatch" pkg/ cmd/`
Expected: no matches.

- [ ] **Step 2: Run the full pre-commit checklist**

Run: `make build && make vet && make lint && make test`
Expected: all pass. (`make lint` now passes because `resolveRLevels`/`countByteDiff` are used by `run`/`download`.)

- [ ] **Step 3: Review the diff against the behavior-preservation checklist**

Run: `git diff master -- pkg/check/smoke/`
Confirm by reading:
- Node selection unchanged: shuffle once, `clients[0]`/`clients[1]`, `≥2` guard.
- Sleeps unchanged: `TxOnErrWait`/`RxOnErrWait` before *every* attempt incl. the first; `NodesSyncWait` between upload and download; `IterationWait` between iterations; `TxOnErrWait` on batch error.
- Retry counts unchanged: 3 upload, 3 download.
- One metric increment per attempt; `attempts == sum over result`.
- Log lines preserved (the only intended change: uploader/downloader name logs now emitted once before the loop instead of every iteration, since the pair is fixed).
- Vestigial fields (`ContentSize`, `UploadTimeout`/`DownloadTimeout`/`IterationWait` wiring) untouched.

- [ ] **Step 4: (Optional) race check**

Run: `make test-race` (or `go test -race ./pkg/check/smoke/...`)
Expected: PASS.

---

## Self-Review notes

- **Spec coverage:** Part A (metrics reshape) → Task 2 Step 1. Part B (decomposition) → Task 2 Step 2. Pure helpers + tests → Task 1. Hard-replace + no leftover names → Task 3 Step 1. Behavior preservation (sleeps, nodes, retries, vestigial fields) → Task 3 Step 3.
- **Type consistency:** `transferrer` matches `pkg/test` method signatures; `*bee.Client` is the `ClientList` element type; `result*` constants used consistently in smoke.go; metric field names (`BatchCreate`/`Upload`/`Download`/...) consistent between metrics.go and smoke.go.
- **No placeholders:** all steps contain full file contents or exact commands.

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

	for range 3 {
		select {
		case <-ctx.Done():
			return swarm.ZeroAddress, false
		case <-time.After(o.TxOnErrWait):
		}

		txCtx, txCancel := context.WithTimeout(ctx, o.UploadTimeout)
		address, txDuration, err := t.Upload(txCtx, uploader, data, batchID, rLevel)
		txCancel()
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

	for range 3 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(o.RxOnErrWait):
		}

		rxCtx, rxCancel := context.WithTimeout(ctx, o.DownloadTimeout)
		data, rxDuration, err := t.Download(rxCtx, downloader, addr, rLevel)
		rxCancel()
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

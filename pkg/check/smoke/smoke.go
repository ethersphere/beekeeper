package smoke

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
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
	"github.com/ethersphere/beekeeper/pkg/topohealth"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// onFailureStorerProbeCount is the number of intended storers to probe on a
	// download failure (closest by XOR distance to root chunk address).
	onFailureStorerProbeCount = 3
	// chunkWalkParallelism caps concurrent HEAD /chunks/{addr} requests during
	// the on-failure chunk walk.
	chunkWalkParallelism = 32
	// chunkWalkMaxReported truncates per-category result lists so a large file
	// with many missing chunks does not flood the log.
	chunkWalkMaxReported = 50
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

// Run creates file of specified size that is uploaded and downloaded.
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
	fileSizes := o.FileSizes
	c.logger.Infof("testing file sizes: %v", fileSizes)
	c.logger.Infof("upload timeout: %s", o.UploadTimeout.String())
	c.logger.Infof("download timeout: %s", o.DownloadTimeout.String())
	c.logger.Infof("total duration: %s", o.Duration.String())

	rnd := random.PseudoGenerator(o.RndSeed)

	// Get shuffled full node clients for better load distribution and testing
	fullNodeClients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("get shuffled full node clients: %w", err)
	}

	if len(fullNodeClients) < 2 {
		return fmt.Errorf("smoke check requires at least 2 full nodes, got %d", len(fullNodeClients))
	}

	shape := topohealth.ClusterShape(cluster)
	c.metrics.ClusterFullNodeCount.Set(float64(shape.FullNodes))
	c.metrics.ClusterLightNodeCount.Set(float64(shape.LightNodes))
	c.logger.Infof("cluster shape: full=%d light=%d bootnodes=%d", shape.FullNodes, shape.LightNodes, shape.Bootnodes)

	thresholds := topohealth.DefaultThresholds()

	test := test.NewTest(c.logger)

	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			c.logger.Infof("starting iteration: #%d", i)
		}

		// Select two different full nodes from the shuffled list
		uploader := fullNodeClients[0]
		downloader := fullNodeClients[1]

		c.logger.Infof("uploader: %s", uploader.Name())
		c.logger.Infof("downloader: %s", downloader.Name())

		c.metrics.BatchCreateAttempts.Inc()

		batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
		if err != nil {
			c.logger.Errorf("create new batch failed: %v", err)
			c.metrics.BatchCreateErrors.Inc()
			c.logger.Infof("retrying in: %v", o.TxOnErrWait)
			time.Sleep(o.TxOnErrWait)
			continue
		}

		c.logger.WithField("batch_id", batchID).Infof("node %s: using batch", uploader.Name())

		rLevels := o.RLevels
		if len(rLevels) == 0 {
			rLevels = []*redundancy.Level{nil}
		}
		rLevelIdx := 0
		for {
			rLevel := rLevels[rLevelIdx]
			for _, contentSize := range fileSizes {
				select {
				case <-ctx.Done():
					return nil
				default:
					if rLevel != nil {
						c.logger.Infof("testing file size: %d bytes (%.2f KB), redundancy level: %d", contentSize, float64(contentSize)/1024, *rLevel)
					} else {
						c.logger.Infof("testing file size: %d bytes (%.2f KB), redundancy level: (not set)", contentSize, float64(contentSize)/1024)
					}
				}

				sizeLabel := fmt.Sprintf("%d", contentSize)
				rLevelLabel := redundancyLevelLabel(rLevel)

				var (
					txDuration time.Duration
					rxDuration time.Duration
					txData     []byte
					rxData     []byte
					address    swarm.Address
					uploaded   bool
					downloaded bool
				)

				txData = make([]byte, contentSize)
				if _, err := rand.Read(txData); err != nil {
					c.logger.Errorf("unable to create random content for size %d: %v", contentSize, err)
					continue
				}

				// Pre-compute the chunk address tree locally so we can pin-point a
				// missing chunk if download later fails. Deterministic for the same
				// (data, rLevel) input — matches what bee would produce.
				splitRoot, allChunks, splitErr := topohealth.SplitChunkAddresses(ctx, txData, rLevel)
				if splitErr != nil {
					c.logger.Errorf("local chunk split failed for size %d: %v", contentSize, splitErr)
					allChunks = nil // fall back to root-only diagnostics
				} else {
					c.logger.Infof("local split produced %d chunks (root=%s)", len(allChunks), splitRoot)
				}

				if c.probe(ctx, topohealth.PhasePreUpload, uploader, thresholds) == topohealth.StatusUnhealthy {
					c.metrics.UnhealthyAbortsPreUp.Inc()
					c.logger.Errorf("aborting iteration: uploader %s is UNHEALTHY pre-upload", uploader.Name())
					continue
				}

				var (
					txCtx    context.Context
					txCancel context.CancelFunc = func() {}
				)

				for range 3 {
					txCancel()

					uploaded = false

					select {
					case <-ctx.Done():
						return nil
					case <-time.After(o.TxOnErrWait):
					}

					txCtx, txCancel = context.WithTimeout(ctx, o.UploadTimeout)

					c.metrics.UploadAttempts.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel).Inc()
					address, txDuration, err = test.Upload(txCtx, uploader, txData, batchID, rLevel)
					if err != nil {
						c.metrics.UploadErrors.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel).Inc()
						c.logger.Errorf("upload failed for size %d: %v", contentSize, err)
						c.logger.Infof("retrying in: %v", o.TxOnErrWait)
					} else {
						uploaded = true
						break
					}
				}
				txCancel()
				if !uploaded {
					c.logger.Infof("skipping download for size %d due to upload failure", contentSize)
					continue
				}

				c.metrics.UploadDuration.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel).Observe(txDuration.Seconds())
				c.metrics.UploadSuccess.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel).Inc()
				c.metrics.UploadedBytes.WithLabelValues(uploader.Name(), rLevelLabel).Add(float64(contentSize))

				if txDuration.Seconds() > 0 {
					uploadThroughput := float64(contentSize) / txDuration.Seconds()
					c.metrics.UploadThroughput.WithLabelValues(sizeLabel, uploader.Name(), rLevelLabel).Set(uploadThroughput)
				}

				time.Sleep(o.NodesSyncWait)

				if c.probe(ctx, topohealth.PhasePreDownload, downloader, thresholds) == topohealth.StatusUnhealthy {
					c.metrics.UnhealthyAbortsPreDown.Inc()
					c.logger.Warningf("downloader %s is UNHEALTHY pre-download; attempting anyway", downloader.Name())
				}

				var (
					rxCtx    context.Context
					rxCancel context.CancelFunc = func() {}
				)

				for range 3 {
					rxCancel()

					select {
					case <-ctx.Done():
						return nil
					case <-time.After(o.RxOnErrWait):
					}

					c.metrics.DownloadAttempts.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Inc()

					rxCtx, rxCancel = context.WithTimeout(ctx, o.DownloadTimeout)
					rxData, rxDuration, err = test.Download(rxCtx, downloader, address, rLevel)
					if err != nil {
						c.metrics.DownloadErrors.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Inc()
						if errors.Is(err, io.ErrUnexpectedEOF) {
							c.metrics.DownloadEOFErrors.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Inc()
						}
						c.logger.Errorf("download failed for size %d: %v", contentSize, err)
						c.logger.Infof("retrying in: %v", o.RxOnErrWait)
						continue
					}

					if bytes.Equal(rxData, txData) {
						c.metrics.DownloadDuration.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Observe(rxDuration.Seconds())
						c.metrics.DownloadSuccess.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Inc()
						c.metrics.DownloadedBytes.WithLabelValues(downloader.Name(), rLevelLabel).Add(float64(contentSize))

						if rxDuration.Seconds() > 0 {
							downloadThroughput := float64(contentSize) / rxDuration.Seconds()
							c.metrics.DownloadThroughput.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Set(downloadThroughput)
						}
						downloaded = true
						break
					}

					c.logger.Infof("data mismatch for size %d: uploaded and downloaded data differ", contentSize)
					c.metrics.DownloadMismatch.WithLabelValues(sizeLabel, downloader.Name(), rLevelLabel).Inc()

					rxLen, txLen := len(rxData), len(txData)
					if rxLen != txLen {
						c.logger.Errorf("length mismatch for size %d: downloaded %d bytes, uploaded %d bytes", contentSize, rxLen, txLen)
						continue
					}

					var diff int
					for i := range txData {
						if txData[i] != rxData[i] {
							diff++
						}
					}
					c.logger.Infof("data mismatch for size %d: found %d different bytes, ~%.2f%%", contentSize, diff, float64(diff)/float64(txLen)*100)
				}
				rxCancel()

				if !downloaded {
					c.logger.Errorf("all download attempts failed for size %d, dumping topology health and walking chunk tree", contentSize)
					c.onFailureDump(ctx, cluster, uploader, downloader, address, allChunks, thresholds)
				}

				c.logger.Infof("completed testing file size: %d bytes", contentSize)
			}
			rLevelIdx++
			if rLevelIdx >= len(rLevels) {
				break
			}
		}

		time.Sleep(o.IterationWait)
	}

	return nil
}

func (c *Check) Report() []prometheus.Collector {
	return c.metrics.Report()
}

func redundancyLevelLabel(rLevel *redundancy.Level) string {
	if rLevel == nil {
		return "not_set"
	}
	return strconv.Itoa(int(*rLevel))
}

// probe runs the per-node topology health probe, emits structured log + metric,
// and returns the resulting Status. A probe error yields StatusUnknown so
// callers can distinguish a transient API failure from a genuinely unhealthy
// node when deciding whether to abort.
func (c *Check) probe(ctx context.Context, phase topohealth.Phase, client *bee.Client, thresholds topohealth.Thresholds) topohealth.Status {
	v, err := topohealth.Probe(ctx, client, thresholds)
	if err != nil {
		c.logger.Errorf("probe %s on %s failed: %v", phase, client.Name(), err)
		c.metrics.NodeHealthVerdict.WithLabelValues(client.Name(), string(phase)).Set(float64(topohealth.StatusUnknown))
		return topohealth.StatusUnknown
	}
	c.metrics.NodeHealthVerdict.WithLabelValues(client.Name(), string(phase)).Set(float64(v.Status))
	topohealth.LogVerdict(c.logger, phase, v)
	return v.Status
}

// onFailureDump fans out four independent diagnostics concurrently: uploader
// probe, downloader probe, the 3 intended-storer probes (with HEAD on the
// root), and the full chunk walk that classifies every missing chunk as
// out-of-AOR (bee#5400 bug 1), in-AOR-not-stored (bug 2/3), or
// cluster-coverage-gap.
func (c *Check) onFailureDump(ctx context.Context, cluster orchestration.Cluster, uploader, downloader *bee.Client, root swarm.Address, allChunks []topohealth.ChunkInfo, thresholds topohealth.Thresholds) {
	var (
		wg      sync.WaitGroup
		storers []topohealth.StorerResult
		storErr error
	)
	wg.Add(4)
	go func() {
		defer wg.Done()
		c.probe(ctx, topohealth.PhaseOnFailure, uploader, thresholds)
	}()
	go func() {
		defer wg.Done()
		c.probe(ctx, topohealth.PhaseOnFailure, downloader, thresholds)
	}()
	go func() {
		defer wg.Done()
		storers, storErr = topohealth.IntendedStorers(ctx, cluster, root, onFailureStorerProbeCount, thresholds)
	}()
	go func() {
		defer wg.Done()
		if len(allChunks) == 0 {
			c.logger.Warningf("on_failure: no pre-computed chunk list (split failed earlier); skipping chunk walk")
			return
		}
		c.walkChunksOnFailure(ctx, cluster, root, allChunks)
	}()
	wg.Wait()

	if storErr != nil {
		c.logger.Errorf("on_failure intended storers probe failed: %v", storErr)
		return
	}
	for i, r := range storers {
		topohealth.LogStorerResult(c.logger, root.String(), string(topohealth.PhaseOnFailure), i, r)
	}
}

// walkChunksOnFailure does a HEAD /chunks/{addr} per chunk against its
// closest full node, records the per-bug counters, and logs every missing or
// out-of-AOR-present chunk.
func (c *Check) walkChunksOnFailure(ctx context.Context, cluster orchestration.Cluster, root swarm.Address, chunks []topohealth.ChunkInfo) {
	storers, err := topohealth.GatherStorers(ctx, cluster)
	if err != nil {
		c.logger.Errorf("on_failure gather storers failed: %v", err)
		return
	}
	res, err := topohealth.WalkChunks(ctx, storers, chunks, chunkWalkParallelism, chunkWalkMaxReported)
	if err != nil {
		// ctx-cancellation is reported here but we still emit partial counters.
		c.logger.Warningf("on_failure chunk walk did not complete: %v", err)
	}

	c.metrics.ChunksChecked.Add(float64(res.Checked))
	for pos, n := range res.MissingTotal {
		c.metrics.ChunksMissingTotal.WithLabelValues(string(pos)).Add(float64(n))
	}
	for pos, n := range res.MissingOutOfAOR {
		c.metrics.ChunksMissingOutOfAOR.WithLabelValues(string(pos)).Add(float64(n))
	}
	for pos, n := range res.MissingInAOR {
		c.metrics.ChunksMissingInAOR.WithLabelValues(string(pos)).Add(float64(n))
	}
	for pos, n := range res.PresentOutOfAOR {
		c.metrics.ChunksPresentOutOfAOR.WithLabelValues(string(pos)).Add(float64(n))
	}

	for _, m := range res.Missing {
		topohealth.LogChunkCheck(c.logger, "missing", root.String(), m)
	}
	for _, p := range res.OutOfAORHits {
		topohealth.LogChunkCheck(c.logger, "present_out_of_aor", root.String(), p)
	}

	totalMissing := sumPerPosition(res.MissingTotal)
	if totalMissing > 0 {
		c.metrics.FilesWithLoss.Inc()
	}
	c.logger.Infof("on_failure walk: root=%s checked=%d probe_errors=%d missing=%d (out_of_aor=%d, in_aor=%d) present_out_of_aor=%d",
		root, res.Checked, res.ProbeErrors, totalMissing,
		sumPerPosition(res.MissingOutOfAOR), sumPerPosition(res.MissingInAOR), sumPerPosition(res.PresentOutOfAOR))
}

func sumPerPosition(m topohealth.PerPositionCounts) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

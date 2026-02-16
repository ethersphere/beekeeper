package smoke

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/scheduler"
	"github.com/ethersphere/beekeeper/pkg/test"
	"github.com/prometheus/client_golang/prometheus"
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

		for _, contentSize := range fileSizes {
			select {
			case <-ctx.Done():
				return nil
			default:
				c.logger.Infof("testing file size: %d bytes (%.2f KB)", contentSize, float64(contentSize)/1024)
			}

			sizeLabel := fmt.Sprintf("%d", contentSize)

			var (
				txDuration time.Duration
				rxDuration time.Duration
				txData     []byte
				rxData     []byte
				address    swarm.Address
				uploaded   bool
			)

			txData = make([]byte, contentSize)
			if _, err := rand.Read(txData); err != nil {
				c.logger.Errorf("unable to create random content for size %d: %v", contentSize, err)
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

				c.metrics.UploadAttempts.WithLabelValues(sizeLabel, uploader.Name()).Inc()
				address, txDuration, err = test.Upload(txCtx, uploader, txData, batchID)
				if err != nil {
					c.metrics.UploadErrors.WithLabelValues(sizeLabel, uploader.Name()).Inc()
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

			c.metrics.UploadDuration.WithLabelValues(sizeLabel, uploader.Name()).Observe(txDuration.Seconds())

			// Calculate and record upload throughput in bytes per second
			if txDuration.Seconds() > 0 {
				uploadThroughput := float64(contentSize) / txDuration.Seconds()
				c.metrics.UploadThroughput.WithLabelValues(sizeLabel, uploader.Name()).Set(uploadThroughput)
			}

			time.Sleep(o.NodesSyncWait)

			var (
				rxCtx      context.Context
				rxCancel   context.CancelFunc = func() {}
				downloaded bool
			)

			for range 3 {
				rxCancel()

				select {
				case <-ctx.Done():
					return nil
				case <-time.After(o.RxOnErrWait):
				}

				c.metrics.DownloadAttempts.WithLabelValues(sizeLabel, downloader.Name()).Inc()

				rxCtx, rxCancel = context.WithTimeout(ctx, o.DownloadTimeout)
				rxData, rxDuration, err = test.Download(rxCtx, downloader, address)
				if err != nil {
					c.metrics.DownloadErrors.WithLabelValues(sizeLabel, downloader.Name()).Inc()
					c.logger.Errorf("download failed for size %d: %v", contentSize, err)
					c.logger.Infof("retrying in: %v", o.RxOnErrWait)
					continue
				}

				// good download
				if bytes.Equal(rxData, txData) {
					c.metrics.DownloadDuration.WithLabelValues(sizeLabel, downloader.Name()).Observe(rxDuration.Seconds())

					if rxDuration.Seconds() > 0 {
						downloadThroughput := float64(contentSize) / rxDuration.Seconds()
						c.metrics.DownloadThroughput.WithLabelValues(sizeLabel, downloader.Name()).Set(downloadThroughput)
					}
					downloaded = true
					break
				}

				// bad download
				c.logger.Infof("data mismatch for size %d: uploaded and downloaded data differ", contentSize)
				c.metrics.DownloadMismatch.WithLabelValues(sizeLabel, downloader.Name()).Inc()

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
				c.logger.Errorf("all download attempts failed for size %d, fetching downloader topology", contentSize)
				top, topErr := downloader.Topology(ctx)
				if topErr != nil {
					c.logger.Errorf("failed to get downloader topology: %v", topErr)
				} else {
					c.logger.Infof("downloader %s topology: depth=%d, connected=%d, population=%d, reachability=%s, bins=%s",
						downloader.Name(), top.Depth, top.Connected, top.Population, top.Reachability, top.Bins.String())
				}
			}

			c.logger.Infof("completed testing file size: %d bytes", contentSize)
		}

		time.Sleep(o.IterationWait)
	}

	return nil
}

func (c *Check) Report() []prometheus.Collector {
	return c.metrics.Report()
}

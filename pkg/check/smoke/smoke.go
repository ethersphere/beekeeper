package smoke

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/scheduler"
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
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) error {
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

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	test := &test{clients: clients, logger: c.logger}

	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			c.logger.Infof("starting iteration: #%d", i)
		}

		perm := rnd.Perm(cluster.Size())
		txIdx := perm[0]
		rxIdx := perm[1]

		// if the upload and download nodes are the same, try again for a different peer
		if txIdx == rxIdx {
			continue
		}

		nn := cluster.NodeNames()
		txName := nn[txIdx]
		rxName := nn[rxIdx]

		c.logger.Infof("uploader: %s", txName)
		c.logger.Infof("downloader: %s", rxName)

		c.metrics.BatchCreateAttempts.Inc()

		batchID, err := clients[txName].GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
		if err != nil {
			c.logger.Errorf("create new batch: %v", err)
			c.metrics.BatchCreateErrors.Inc()
			continue
		}

		c.logger.WithField("batch_id", batchID).Infof("using batch")

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
				c.logger.Infof("unable to create random content for size %d: %v", contentSize, err)
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

				c.logger.WithField("batch_id", batchID).Infof("using batch for size %d", contentSize)

				c.metrics.UploadAttempts.WithLabelValues(sizeLabel).Inc()
				address, txDuration, err = test.upload(txCtx, txName, txData, batchID)
				if err != nil {
					c.metrics.UploadErrors.WithLabelValues(sizeLabel).Inc()
					c.logger.Infof("upload failed for size %d: %v", contentSize, err)
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

			c.metrics.UploadDuration.WithLabelValues(sizeLabel).Observe(txDuration.Seconds())

			// Calculate and record upload throughput in bytes per second
			if txDuration.Seconds() > 0 {
				uploadThroughput := float64(contentSize) / txDuration.Seconds()
				c.metrics.UploadThroughput.WithLabelValues(sizeLabel).Set(uploadThroughput)
			}
			c.logger.Infof("upload completed for size %d in %s", contentSize, txDuration)

			time.Sleep(o.NodesSyncWait)

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

				c.metrics.DownloadAttempts.WithLabelValues(sizeLabel).Inc()

				rxCtx, rxCancel = context.WithTimeout(ctx, o.DownloadTimeout)
				rxData, rxDuration, err = test.download(rxCtx, rxName, address)
				if err != nil {
					c.metrics.DownloadErrors.WithLabelValues(sizeLabel).Inc()
					c.logger.Infof("download failed for size %d: %v", contentSize, err)
					c.logger.Infof("retrying in: %v", o.RxOnErrWait)
					continue
				}

				// good download
				if bytes.Equal(rxData, txData) {
					c.metrics.DownloadDuration.WithLabelValues(sizeLabel).Observe(rxDuration.Seconds())

					if rxDuration.Seconds() > 0 {
						downloadThroughput := float64(contentSize) / rxDuration.Seconds()
						c.metrics.DownloadThroughput.WithLabelValues(sizeLabel).Set(downloadThroughput)
					}
					c.logger.Infof("download completed for size %d in %s", contentSize, rxDuration)
					break
				}

				// bad download
				c.logger.Infof("uploaded data does not match downloaded data for size %d", contentSize)
				c.metrics.DownloadMismatch.WithLabelValues(sizeLabel).Inc()

				rxLen, txLen := len(rxData), len(txData)
				if rxLen != txLen {
					c.logger.Infof("length mismatch for size %d: download length %d; upload length %d", contentSize, rxLen, txLen)
					if txLen < rxLen {
						c.logger.Infof("length mismatch for size %d: rx length is bigger than tx length", contentSize)
					}
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
			c.logger.Infof("completed testing file size: %d bytes", contentSize)
		}

		time.Sleep(o.IterationWait)
	}

	return nil
}

type test struct {
	clients map[string]*bee.Client
	logger  logging.Logger
}

func (t *test) upload(ctx context.Context, cName string, data []byte, batchID string) (swarm.Address, time.Duration, error) {
	client := t.clients[cName]
	t.logger.Infof("node %s: uploading data, batch id %s", cName, batchID)
	start := time.Now()
	addr, err := client.UploadBytes(ctx, data, api.UploadOptions{Pin: false, BatchID: batchID, Direct: true})
	if err != nil {
		return swarm.ZeroAddress, 0, fmt.Errorf("upload to the node %s: %w", cName, err)
	}
	txDuration := time.Since(start)
	t.logger.Infof("node %s: upload done in %s", cName, txDuration)

	return addr, txDuration, nil
}

func (t *test) download(ctx context.Context, cName string, addr swarm.Address) ([]byte, time.Duration, error) {
	client := t.clients[cName]
	t.logger.Infof("node %s: downloading address %s", cName, addr)
	start := time.Now()
	data, err := client.DownloadBytes(ctx, addr, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("download from node %s: %w", cName, err)
	}
	rxDuration := time.Since(start)
	t.logger.Infof("node %s: download done in %s", cName, rxDuration)

	return data, rxDuration, nil
}

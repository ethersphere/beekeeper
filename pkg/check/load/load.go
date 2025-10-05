package load

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/check/smoke"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/scheduler"
	"github.com/ethersphere/beekeeper/pkg/test"
	"github.com/prometheus/client_golang/prometheus"
)

type Options struct {
	ContentSize             int64
	RndSeed                 int64
	PostageTTL              time.Duration
	PostageDepth            uint64
	PostageLabel            string
	TxOnErrWait             time.Duration
	RxOnErrWait             time.Duration
	NodesSyncWait           time.Duration
	Duration                time.Duration
	UploaderCount           int
	UploadGroups            []string
	DownloaderCount         int
	DownloadGroups          []string
	MaxCommittedDepth       uint8
	CommittedDepthCheckWait time.Duration
	IterationWait           time.Duration
}

func NewDefaultOptions() Options {
	return Options{
		ContentSize:             5000000,
		RndSeed:                 time.Now().UnixNano(),
		PostageTTL:              24 * time.Hour,
		PostageDepth:            24,
		PostageLabel:            "test-label",
		TxOnErrWait:             10 * time.Second,
		RxOnErrWait:             10 * time.Second,
		NodesSyncWait:           time.Minute,
		Duration:                12 * time.Hour,
		UploaderCount:           1,
		UploadGroups:            []string{"bee"},
		DownloaderCount:         0,
		DownloadGroups:          []string{},
		MaxCommittedDepth:       2,
		CommittedDepthCheckWait: 5 * time.Minute,
		IterationWait:           5 * time.Minute,
	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

var _ beekeeper.Action = (*Check)(nil)

type Check struct {
	metrics smoke.Metrics
	logger  logging.Logger
}

func NewCheck(log logging.Logger) beekeeper.Action {
	return &Check{
		metrics: smoke.NewMetrics("check_load"),
		logger:  log,
	}
}

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
	if o.UploaderCount == 0 || len(o.UploadGroups) == 0 {
		return errors.New("no uploaders requested, quitting")
	}

	if o.MaxCommittedDepth == 0 {
		return errors.New("max committed depth is not set")
	}

	contentSize := o.ContentSize

	c.logger.Infof("random seed: %v", o.RndSeed)
	c.logger.Infof("content size: %v", contentSize)
	c.logger.Infof("max committed depth: %v", o.MaxCommittedDepth)
	c.logger.Infof("committed depth check wait time: %v", o.CommittedDepthCheckWait)
	c.logger.Infof("total duration: %s", o.Duration.String())

	rnd := random.PseudoGenerator(o.RndSeed)
	fullNodeClients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("get shuffled full node clients: %w", err)
	}

	minNodes := min(o.UploaderCount, o.DownloaderCount)
	if len(fullNodeClients) == 0 || len(fullNodeClients) < minNodes {
		return fmt.Errorf("load check requires at least %d full nodes, got %d", minNodes, len(fullNodeClients))
	}

	test := test.NewTest(c.logger)

	var downloaders []*bee.Client
	if o.DownloaderCount > 0 && len(o.DownloadGroups) > 0 {
		downloaders = fullNodeClients.FilterByNodeGroups(o.DownloadGroups)
		if len(downloaders) == 0 {
			return fmt.Errorf("no downloaders found in the specified node groups: %v", o.DownloadGroups)
		}
		if len(downloaders) < o.DownloaderCount {
			return fmt.Errorf("not enough downloaders found in the specified node groups: %v, requested %d, got %d", o.DownloadGroups, o.DownloaderCount, len(downloaders))
		}
	}

	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			c.logger.Info("context done in iteration")
			return nil
		default:
			c.logger.Infof("starting iteration: #%d bytes (%.2f KB)", contentSize, float64(contentSize)/1024)
		}

		sizeLabel := fmt.Sprintf("%d", contentSize)

		var (
			txDuration time.Duration
			txData     []byte
			address    swarm.Address
		)

		txData = make([]byte, contentSize)
		if _, err := crand.Read(txData); err != nil {
			c.logger.Infof("unable to create random content for size %d: %v", contentSize, err)
			continue
		}

		uploaders := fullNodeClients
		if o.UploaderCount > 0 && len(o.UploadGroups) > 0 {
			uploaders = fullNodeClients.FilterByNodeGroups(o.UploadGroups)
			if len(uploaders) == 0 {
				return fmt.Errorf("no uploaders found in the specified node groups: %v", o.UploadGroups)
			}
			if len(uploaders) < o.UploaderCount {
				return fmt.Errorf("not enough uploaders found in the specified node groups: %v, requested %d, got %d", o.UploadGroups, o.UploaderCount, len(uploaders))
			}
		}

		var (
			upload sync.WaitGroup
			once   sync.Once
		)

		upload.Add(1)

		for _, uploader := range uploaders[:o.UploaderCount] {
			go func() {
				defer once.Do(func() {
					upload.Done()
				}) // don't wait for all uploads
				for retries := 10; txDuration == 0 && retries > 0; retries-- {
					select {
					case <-ctx.Done():
						c.logger.Info("context done in retry")
						return
					default:
					}

					if !c.checkCommittedDepth(ctx, uploader, o.MaxCommittedDepth, o.CommittedDepthCheckWait) {
						return
					}

					c.metrics.UploadAttempts.WithLabelValues(sizeLabel).Inc()
					var duration time.Duration
					c.logger.Infof("uploading to: %s", uploader)

					batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
					if err != nil {
						c.logger.Errorf("create new batch: %v", err)
						return
					}

					c.logger.WithField("batch_id", batchID).Info("using batch")

					address, duration, err = test.Upload(ctx, uploader, txData, batchID)
					if err != nil {
						c.metrics.UploadErrors.WithLabelValues(sizeLabel).Inc()
						c.logger.Infof("upload failed: %v", err)
						c.logger.Infof("retrying in: %v", o.TxOnErrWait)
						time.Sleep(o.TxOnErrWait)
						return
					}
					txDuration += duration // dirty
				}
			}()
		}

		upload.Wait()

		if txDuration == 0 {
			continue
		}

		c.logger.Infof("sleeping for: %v seconds", o.NodesSyncWait.Seconds())
		time.Sleep(o.NodesSyncWait)

		var wg sync.WaitGroup

		for _, downloader := range downloaders {
			wg.Add(1)
			go func() {
				defer wg.Done()

				var (
					rxDuration time.Duration
					rxData     []byte
				)

				for retries := 10; rxDuration == 0 && retries > 0; retries-- {
					select {
					case <-ctx.Done():
						c.logger.Infof("context done in retry: %v", retries)
						return
					default:
					}

					c.metrics.DownloadAttempts.WithLabelValues(sizeLabel).Inc()

					rxData, rxDuration, err = test.Download(ctx, downloader, address)
					if err != nil {
						c.metrics.DownloadErrors.WithLabelValues(sizeLabel).Inc()
						c.logger.Infof("download failed: %v", err)
						c.logger.Infof("retrying in: %v", o.RxOnErrWait)
						time.Sleep(o.RxOnErrWait)
					}
				}

				if rxDuration == 0 {
					return
				}

				if !bytes.Equal(rxData, txData) {
					c.logger.Info("uploaded data does not match downloaded data")

					c.metrics.DownloadMismatch.WithLabelValues(sizeLabel).Inc()

					rxLen, txLen := len(rxData), len(txData)
					if rxLen != txLen {
						c.logger.Infof("length mismatch: download length %d; upload length %d", rxLen, txLen)
						if txLen < rxLen {
							c.logger.Info("length mismatch: rx length is bigger than tx length")
						}
						return
					}

					var diff int
					for i := range txData {
						if txData[i] != rxData[i] {
							diff++
						}
					}
					c.logger.Infof("data mismatch: found %d different bytes, ~%.2f%%", diff, float64(diff)/float64(txLen)*100)
					return
				}

				c.metrics.UploadDuration.WithLabelValues(sizeLabel).Observe(txDuration.Seconds())
				c.metrics.DownloadDuration.WithLabelValues(sizeLabel).Observe(rxDuration.Seconds())
				if txDuration.Seconds() > 0 {
					uploadThroughput := float64(contentSize) / txDuration.Seconds()
					c.metrics.UploadThroughput.WithLabelValues(sizeLabel).Set(uploadThroughput)
				}
				if rxDuration.Seconds() > 0 {
					downloadThroughput := float64(contentSize) / rxDuration.Seconds()
					c.metrics.DownloadThroughput.WithLabelValues(sizeLabel).Set(downloadThroughput)
				}
			}()
		}

		wg.Wait()
	}

	return nil
}

func (c *Check) checkCommittedDepth(ctx context.Context, client *bee.Client, maxDepth uint8, wait time.Duration) bool {
	for {
		statusResp, err := client.Status(ctx)
		if err != nil {
			c.logger.Infof("error getting state: %v", err)
			return false
		}

		if statusResp.CommittedDepth < maxDepth {
			return true
		}
		c.logger.Infof("waiting %v for CommittedDepth to decrease. Current: %d, Max: %d", wait, statusResp.CommittedDepth, maxDepth)

		select {
		case <-ctx.Done():
			c.logger.Infof("context done while waiting for CommittedDepth to decrease: %v", ctx.Err())
			return false
		case <-time.After(wait):
		}
	}
}

func (c *Check) Report() []prometheus.Collector {
	return c.metrics.Report()
}

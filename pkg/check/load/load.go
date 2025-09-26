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
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/scheduler"
)

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

var _ beekeeper.Action = (*Check)(nil)

type Check struct {
	metrics metrics
	logger  logging.Logger
}

func NewCheck(log logging.Logger) beekeeper.Action {
	return &Check{
		metrics: newMetrics("check_load"),
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

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("get clients: %w", err)
	}

	test := &test{clients: clients, logger: c.logger}

	uploaders := selectNames(cluster, o.UploadGroups...)
	downloaders := selectNames(cluster, o.DownloadGroups...)

	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			c.logger.Info("we are done")
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

		txNames := pickRandom(o.UploaderCount, uploaders)

		c.logger.Infof("uploader: %s", txNames)

		var (
			upload sync.WaitGroup
			once   sync.Once
		)

		upload.Add(1)
	
		for _, txName := range txNames {
			go func() {
				defer once.Do(func() {
					upload.Done()
				})
				for retries := 10; txDuration == 0 && retries > 0; retries-- {
					select {
					case <-ctx.Done():
						c.logger.Info("we are done")
						return
					default:
					}

					if !c.checkCommittedDepth(ctx, test.clients[txName], o.MaxCommittedDepth, o.CommittedDepthCheckWait) {
						return
					}

					c.metrics.UploadAttempts.WithLabelValues(sizeLabel).Inc()
					var duration time.Duration
					c.logger.Infof("uploading to: %s", txName)

					batchID, err := clients[txName].GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
					if err != nil {
						c.logger.Errorf("create new batch: %v", err)
						return
					}

					c.logger.WithField("batch_id", batchID).Info("using batch")

					address, duration, err = test.upload(ctx, txName, txData, batchID)
					if err != nil {
						c.metrics.UploadErrors.WithLabelValues(sizeLabel).Inc()
						c.logger.Infof("upload failed: %v", err)
						c.logger.Infof("retrying in: %v", o.TxOnErrWait)
						time.Sleep(o.TxOnErrWait)
						return
					}
					txDuration += duration
				}
			}()
		}

		upload.Wait()

		if txDuration == 0 {
			continue
		}

		c.logger.Infof("sleeping for: %v seconds", o.NodesSyncWait.Seconds())
		time.Sleep(o.NodesSyncWait)

		rxNames := pickRandom(o.DownloaderCount, downloaders)
		c.logger.Infof("downloaders: %s", rxNames)

		var wg sync.WaitGroup

		for _, rxName := range rxNames {
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

					rxData, rxDuration, err = test.download(ctx, rxName, address)
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

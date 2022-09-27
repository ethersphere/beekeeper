package smoke

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// compile check whether Check implements interface
var _ beekeeper.Action = (*LoadCheck)(nil)

// Check instance
type LoadCheck struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns new check
func NewLoadCheck(logger logging.Logger) beekeeper.Action {
	return &LoadCheck{
		metrics: newMetrics("check_load"),
		logger:  logger,
	}
}

// Run creates file of specified size that is uploaded and downloaded.
func (c *LoadCheck) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	c.logger.Info("random seed: ", o.RndSeed)
	c.logger.Info("content size: ", o.ContentSize)

	rnd := random.PseudoGenerator(o.RndSeed)

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, o.Duration)
	defer cancel()

	test := &test{opt: o, ctx: ctx, clients: clients, logger: c.logger}

	uploaders := selectNames(cluster, o.UploadGroup...)
	downloaders := selectNames(cluster, o.DownloadGroup...)

	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			c.logger.Infof("starting iteration: #%d", i)
		}

		txNames := pick(1, uploaders, cluster, rnd) // for now we only upload to one node
		rxNames := pick(o.DownloaderCount, downloaders, cluster, rnd)

		c.logger.Infof("uploader: %s", txNames)
		c.logger.Infof("downloaders: %s", rxNames)

		var (
			txDuration time.Duration
			txData     []byte
			address    swarm.Address
		)

		txData = make([]byte, o.ContentSize)
		if _, err := rand.Read(txData); err != nil {
			c.logger.Infof("unable to create random content: %v", err)
			continue
		}

		txName := txNames[0]

		for retries := 10; txDuration == 0 && retries > 0; retries-- {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			c.metrics.UploadAttempts.Inc()

			address, txDuration, err = test.upload(txName, txData)
			if err != nil {
				c.metrics.UploadErrors.Inc()
				c.logger.Infof("upload failed: %v", err)
				c.logger.Infof("retrying in: %v", o.TxOnErrWait)
				time.Sleep(o.TxOnErrWait)
			}
		}

		if txDuration == 0 {
			continue
		}

		time.Sleep(o.NodesSyncWait) // Wait for nodes to sync.

		var wg sync.WaitGroup

		for _, rxName := range rxNames {
			rxName := rxName
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

					c.metrics.DownloadAttempts.Inc()

					rxData, rxDuration, err = test.download(rxName, address)
					if err != nil {
						c.metrics.DownloadErrors.Inc()
						c.logger.Infof("download failed: %v", err)
						c.logger.Infof("retrying in: %v", o.RxOnErrWait)
						time.Sleep(o.RxOnErrWait)
					}
				}

				// download error, skip comprarison below
				if rxDuration == 0 {
					return
				}

				if !bytes.Equal(rxData, txData) {
					c.logger.Info("uploaded data does not match downloaded data")

					c.metrics.DownloadMismatch.Inc()

					rxLen, txLen := len(rxData), len(txData)
					if rxLen != txLen {
						c.logger.Infof("length mismatch: download length %d; upload length %d", rxLen, txLen)
						if txLen < rxLen {
							c.logger.Info("length mismatch: rx length is bigger then tx length")
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

				// We want to update the metrics when no error has been
				// encountered in order to avoid counter mismatch.
				c.metrics.UploadDuration.Observe(txDuration.Seconds())
				c.metrics.DownloadDuration.Observe(rxDuration.Seconds())
			}()
		}
		wg.Wait()
	}

	return nil
}

func pick(count int, nn []string, cluster orchestration.Cluster, rnd *rand.Rand) (names []string) {
	for i := 0; i < count; i++ {
	RNG:
		index := rnd.Intn(len(nn))
		name := nn[index]
		if contains(names, name) {
			goto RNG
		}
		names = append(names, name)
	}

	return
}

func contains(in []string, target string) bool {
	for _, name := range in {
		if name == target {
			return true
		}
	}

	return false
}

func selectNames(c orchestration.Cluster, names ...string) (selected []string) {
	for _, name := range names {
		ng, err := c.NodeGroup(name)
		if err != nil {
			panic(err)
		}
		selected = append(selected, ng.NodesSorted()...)
	}

	return
}

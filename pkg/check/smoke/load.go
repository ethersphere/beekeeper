package smoke

import (
	"bytes"
	"context"
	"errors"
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
		return errors.New("invalid options type")
	}

	if o.UploaderCount == 0 || len(o.UploadGroups) == 0 {
		return errors.New("no uploaders requested, quiting")
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

	uploaders := selectNames(cluster, o.UploadGroups...)
	downloaders := selectNames(cluster, o.DownloadGroups...)

	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			c.logger.Info("we are done")
			return nil
		default:
			c.logger.Infof("starting iteration: #%d", i)
		}

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

		txNames := pickRandom(o.UploaderCount, uploaders, rnd)

		c.logger.Infof("uploader: %s", txNames)

		var (
			upload sync.WaitGroup
			once   sync.Once
		)
		upload.Add(1)

		for _, txName := range txNames {
			txName := txName
			go func() {
				defer once.Do(func() { upload.Done() }) // don't wait for all uploads
				for retries := 10; txDuration == 0 && retries > 0; retries-- {
					select {
					case <-ctx.Done():
						c.logger.Info("we are done")
						return
					default:
					}

					c.metrics.UploadAttempts.Inc()
					var duration time.Duration
					c.logger.Infof("uploading to: %s", txName)
					address, duration, err = test.upload(txName, txData)
					if err != nil {
						c.metrics.UploadErrors.Inc()
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
		time.Sleep(o.NodesSyncWait) // Wait for nodes to sync.

		// pick a batch of downloaders
		rxNames := pickRandom(o.DownloaderCount, downloaders, rnd)
		c.logger.Infof("downloaders: %s", rxNames)

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

func pickRandom(count int, peers []string, rnd *rand.Rand) (names []string) {
	seq := randomIntSeq(count, len(peers), rnd)
	for i := range seq {
		names = append(names, peers[i])
	}

	return
}

func selectNames(c orchestration.Cluster, names ...string) (selected []string) {
	for _, name := range names {
		ng, err := c.NodeGroup(name)
		if err != nil {
			panic(err)
		}
		selected = append(selected, ng.NodesSorted()...)
	}

	rand.Shuffle(len(selected), func(i, j int) {
		tmp := selected[i]
		selected[i] = selected[j]
		selected[j] = tmp
	})

	return
}

func randomIntSeq(size, ceiling int, rnd *rand.Rand) (out []int) {
	r := make(map[int]struct{}, size)

	for len(r) < size {
		r[rnd.Intn(ceiling)] = struct{}{}
	}

	for k := range r {
		out = append(out, k)
	}

	return
}

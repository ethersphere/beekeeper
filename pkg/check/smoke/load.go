package smoke

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*LoadCheck)(nil)

// Check instance
type LoadCheck struct {
	metrics metrics
	log     logging.Logger
}

// NewCheck returns new check
func NewLoadCheck(log logging.Logger) beekeeper.Action {
	return &LoadCheck{
		metrics: newMetrics("check_load"),
		log:     log,
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

	if o.MaxStorageRadius == 0 {
		return errors.New("max storage radius is not set")
	}

	c.log.Infof("random seed: %v", o.RndSeed)
	c.log.Infof("content size: %v", o.ContentSize)
	c.log.Infof("max batch lifespan: %v", o.MaxUseBatch)
	c.log.Infof("max storage radius: %v", o.MaxStorageRadius)
	c.log.Infof("storage radius check wait time: %v", o.StorageRadiusCheckWait)

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, o.Duration)
	defer cancel()

	test := &test{clients: clients, logger: c.log}

	uploaders := selectNames(cluster, o.UploadGroups...)
	downloaders := selectNames(cluster, o.DownloadGroups...)

	batches := NewStore(o.MaxUseBatch)

	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			c.log.Info("we are done")
			return nil
		default:
			c.log.Infof("starting iteration: #%d", i)
		}

		var (
			txDuration time.Duration
			txData     []byte
			address    swarm.Address
		)

		txData = make([]byte, o.ContentSize)
		if _, err := crand.Read(txData); err != nil {
			c.log.Infof("unable to create random content: %v", err)
			continue
		}

		txNames := pickRandom(o.UploaderCount, uploaders)

		c.log.Infof("uploader: %s", txNames)

		var (
			upload sync.WaitGroup
			once   sync.Once
		)

		upload.Add(1)

		for _, txName := range txNames {
			txName := txName

			go func() {
				defer once.Do(func() {
					upload.Done()
				}) // don't wait for all uploads
				for retries := 10; txDuration == 0 && retries > 0; retries-- {
					select {
					case <-ctx.Done():
						c.log.Info("we are done")
						return
					default:
					}

					if !c.checkStorageRadius(ctx, test.clients[txName], o.MaxStorageRadius, o.StorageRadiusCheckWait) {
						return
					}

					c.metrics.UploadAttempts.Inc()
					var duration time.Duration
					c.log.Infof("uploading to: %s", txName)

					batchID := batches.Get(txName)
					if batchID == "" {
						batchID, err = clients[txName].CreatePostageBatch(ctx, o.PostageAmount, o.PostageDepth, "load-test", true)
						if err != nil {
							c.log.Errorf("create new batch: %v", err)
							return
						}
						batches.Store(txName, batchID)
					}

					address, duration, err = test.upload(ctx, txName, txData, batchID)
					if err != nil {
						c.metrics.UploadErrors.Inc()
						c.log.Infof("upload failed: %v", err)
						c.log.Infof("retrying in: %v", o.TxOnErrWait)
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

		c.log.Infof("sleeping for: %v seconds", o.NodesSyncWait.Seconds())
		time.Sleep(o.NodesSyncWait) // Wait for nodes to sync.

		// pick a batch of downloaders
		rxNames := pickRandom(o.DownloaderCount, downloaders)
		c.log.Infof("downloaders: %s", rxNames)

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
						c.log.Infof("context done in retry: %v", retries)
						return
					default:
					}

					c.metrics.DownloadAttempts.Inc()

					rxData, rxDuration, err = test.download(ctx, rxName, address)
					if err != nil {
						c.metrics.DownloadErrors.Inc()
						c.log.Infof("download failed: %v", err)
						c.log.Infof("retrying in: %v", o.RxOnErrWait)
						time.Sleep(o.RxOnErrWait)
					}
				}

				// download error, skip comprarison below
				if rxDuration == 0 {
					return
				}

				if !bytes.Equal(rxData, txData) {
					c.log.Info("uploaded data does not match downloaded data")

					c.metrics.DownloadMismatch.Inc()

					rxLen, txLen := len(rxData), len(txData)
					if rxLen != txLen {
						c.log.Infof("length mismatch: download length %d; upload length %d", rxLen, txLen)
						if txLen < rxLen {
							c.log.Info("length mismatch: rx length is bigger then tx length")
						}
						return
					}

					var diff int
					for i := range txData {
						if txData[i] != rxData[i] {
							diff++
						}
					}
					c.log.Infof("data mismatch: found %d different bytes, ~%.2f%%", diff, float64(diff)/float64(txLen)*100)
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

func (c *LoadCheck) checkStorageRadius(ctx context.Context, client *bee.Client, maxRadius uint8, wait time.Duration) bool {
	for {
		rs, err := client.ReserveState(ctx)
		if err != nil {
			c.log.Infof("error getting state: %v", err)
			return false
		}
		if rs.StorageRadius < maxRadius {
			return true
		}
		c.log.Infof("waiting %v for StorageRadius to decrease. Current: %d, Max: %d", wait, rs.StorageRadius, maxRadius)

		select {
		case <-ctx.Done():
			c.log.Infof("context done in StorageRadius check: %v", ctx.Err())
			return false
		case <-time.After(wait):
		}
	}
}

func pickRandom(count int, peers []string) (names []string) {
	seq := randomIntSeq(count, len(peers))
	for _, i := range seq {
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

func randomIntSeq(size, ceiling int) (out []int) {
	r := make(map[int]struct{}, size)

	for len(r) < size {
		r[rand.Intn(ceiling)] = struct{}{}
	}

	for k := range r {
		out = append(out, k)
	}

	return
}

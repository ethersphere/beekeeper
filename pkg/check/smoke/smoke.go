package smoke

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents smoke test options
type Options struct {
	ContentSize   int64
	RndSeed       int64
	PostageAmount int64
	PostageDepth  uint64
	TxOnErrWait   time.Duration
	RxOnErrWait   time.Duration
	NodesSyncWait time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ContentSize:   5000000,
		RndSeed:       0,
		PostageAmount: 1000000,
		PostageDepth:  20,
		TxOnErrWait:   10 * time.Second,
		RxOnErrWait:   10 * time.Second,
		NodesSyncWait: time.Minute,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{newMetrics()}
}

// Run creates file of specified size that is uploaded and downloaded.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	fmt.Println("random seed: ", o.RndSeed)
	fmt.Println("content size: ", o.ContentSize)

	rnd := random.PseudoGenerator(o.RndSeed)

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second) // Wait for the nodes to warmup.

	// The test will restart itself every 12 hours, this is in order to
	// create more meaningful metrics, so that we can apply prometheus
	// functions to them.
	ctx, cancel := context.WithTimeout(ctx, 12*time.Hour)
	defer cancel()

	test := &test{opt: o, ctx: ctx, clients: clients}

	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			fmt.Printf("starting iteration: #%d\n", i)
		}

		perm := rnd.Perm(cluster.Size())
		txIdx := perm[0]
		rxIdx := perm[1]

		if txIdx == rxIdx {
			continue
		}

		nn := cluster.NodeNames()
		txName := nn[txIdx]
		rxName := nn[rxIdx]

		fmt.Printf("uploader: %s\n", txName)
		fmt.Printf("downloader: %s\n", rxName)

		var (
			txDuration time.Duration
			rxDuration time.Duration
			txData     []byte
			rxData     []byte
			address    swarm.Address
		)

		txData = make([]byte, o.ContentSize)
		rnd.Read(txData)

		for txDuration == 0 {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			address, txDuration, err = test.upload(txName, txData)
			if err != nil {
				c.metrics.UploadErrors.Inc()
				fmt.Printf("upload failed: %v\n", err)
				fmt.Printf("retrying in: %v\n", o.TxOnErrWait)
				time.Sleep(o.TxOnErrWait)
			}
		}

		time.Sleep(o.NodesSyncWait) // Wait for nodes to sync.

		for rxDuration == 0 {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			rxData, rxDuration, err = test.download(rxName, address)
			if err != nil {
				c.metrics.DownloadErrors.Inc()
				fmt.Printf("download failed: %v\n", err)
				fmt.Printf("retrying in: %v\n", o.RxOnErrWait)
				time.Sleep(o.RxOnErrWait)
			}
		}

		if !bytes.Equal(rxData, txData) {
			c.metrics.DownloadErrors.Inc()
			fmt.Println("uploaded data does not match downloaded data")
			continue
		}

		// We want to update the metrics when no error has been
		// encountered in order to avoid counter mismatch.
		c.metrics.UploadDuration.Observe(txDuration.Seconds())
		c.metrics.DownloadDuration.Observe(rxDuration.Seconds())
	}

	fmt.Println("smoke test completed successfully")
	return nil
}

type test struct {
	opt     Options
	ctx     context.Context
	clients map[string]*bee.Client
}

func (t *test) upload(cName string, data []byte) (swarm.Address, time.Duration, error) {
	client := t.clients[cName]
	batchID, err := client.GetOrCreateBatch(t.ctx, t.opt.PostageAmount, t.opt.PostageDepth, "", "smoke-test")
	if err != nil {
		return swarm.ZeroAddress, 0, fmt.Errorf("node %s: unable to create batch id: %w", cName, err)
	}
	fmt.Printf("node %s: batch id %s\n", cName, batchID)

	fmt.Printf("node %s: uploading data\n", cName)
	start := time.Now()
	addr, err := client.UploadBytes(t.ctx, data, api.UploadOptions{Pin: false, BatchID: batchID, Deferred: false})
	if err != nil {
		return swarm.ZeroAddress, 0, fmt.Errorf("upload to the node %s: %w", cName, err)
	}
	txDuration := time.Since(start)
	fmt.Printf("node %s: upload done in %s\n", cName, txDuration)

	return addr, txDuration, nil
}

func (t *test) download(cName string, addr swarm.Address) ([]byte, time.Duration, error) {
	client := t.clients[cName]
	fmt.Printf("node %s: downloading data\n", cName)
	start := time.Now()
	data, err := client.DownloadBytes(t.ctx, addr)
	if err != nil {
		return nil, 0, fmt.Errorf("download from node %s: %w", cName, err)
	}
	rxDuration := time.Since(start)
	fmt.Printf("node %s: download done in %s\n", cName, rxDuration)

	return data, rxDuration, nil
}

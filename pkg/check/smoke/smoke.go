package smoke

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/google/uuid"
)

// Options represents smoke test options
type Options struct {
	ContentSize   int64
	Iterations    int
	RndSeed       int64
	GasPrice      string
	PostageAmount int64
	PostageDepth  uint64
	PostageLabel  string
	SyncTimeout   time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ContentSize:   5000000,
		Iterations:    1,
		RndSeed:       0,
		GasPrice:      "",
		PostageAmount: 1000,
		PostageDepth:  16,
		PostageLabel:  "test-label",
		SyncTimeout:   time.Minute,
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
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
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
	for i := 0; i < o.Iterations; i++ {
		perm := rnd.Perm(cluster.Size())
		txIdx := perm[0]
		rxIdx := perm[1]

		if txIdx == rxIdx {
			fmt.Println("warning: uploading node is the same as downloading node!")
		}

		nn := cluster.NodeNames()
		txName := nn[txIdx]
		rxName := nn[rxIdx]
		txClient := clients[txName]
		rxClient := clients[rxName]

		batchID, err := txClient.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: unable to create batch id: %w", txName, err)
		}
		fmt.Printf("#%d: node %s: batch id %s\n", i, txName, batchID)

		file := bee.NewRandomFile(rnd, fmt.Sprintf("check_smoke-#%d-%s", i, uuid.New().String()), o.ContentSize)

		fmt.Printf("#%d: uploading file %s to the node: %s\n", i, file.Name(), txName)
		start := time.Now()
		err = txClient.UploadFile(ctx, &file, api.UploadOptions{Pin: false, BatchID: batchID})
		if err != nil {
			return fmt.Errorf("upload to the node %s: %w", txName, err)
		}
		txDuration := time.Since(start)
		fmt.Printf("#%d: upload done in %s\n", i, txDuration)

		time.Sleep(o.SyncTimeout) // Wait for nodes to sync.

		fmt.Printf("#%d: downloading file %s from the node: %s\n", i, file.Name(), rxName)
		start = time.Now()
		size, hash, err := rxClient.DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("download from node %s: %w", rxName, err)
		}
		rxDuration := time.Since(start)
		fmt.Printf("#%d: download done in %s\n", i, rxDuration)

		if size != file.Size() {
			return errors.New("downloaded file size mismatch")
		}

		if !bytes.Equal(hash, file.Hash()) {
			return errors.New("downloaded file hash mismatch")
		}

		// We want to update the metrics when no error has been
		// encountered in order to avoid counter mismatch.
		c.metrics.Iterations.Inc()
		c.metrics.FileUploadDuration.Set(txDuration.Seconds())
		c.metrics.FileDownloadDuration.Set(rxDuration.Seconds())
	}

	fmt.Println("smoke test completed successfully")
	return nil
}

package smoke

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents smoke test options
type Options struct {
	ContentSize   int
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
		ContentSize:   5 * 1024 << 10,
		Iterations:    1,
		RndSeed:       0,
		GasPrice:      "",
		PostageAmount: 1000,
		PostageDepth:  16,
		PostageLabel:  "test-label",
		SyncTimeout:   time.Second,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
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

		tr, err := txClient.CreateTag(ctx)
		if err != nil {
			return fmt.Errorf("get tag from node %s: %w", txName, err)
		}

		batchID, err := txClient.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: unable to create batch id: %w", txName, err)
		}
		fmt.Printf("node %s: batch id %s\n", txName, batchID)

		data := make([]byte, o.ContentSize)
		if _, err := rnd.Read(data); err != nil {
			return fmt.Errorf("create random data: %w", err)
		}

		fmt.Printf("#%d: uplading to the node: %s\n", i, txName)
		addr, err := txClient.UploadBytes(ctx, data, api.UploadOptions{Pin: false, Tag: tr.Uid, BatchID: batchID})
		if err != nil {
			return fmt.Errorf("upload to the node %s: %w", txName, err)
		}

		time.Sleep(5 * time.Second) // Wait for nodes to sync.

		sCtx, cancel := context.WithTimeout(ctx, o.SyncTimeout)
		defer cancel()

		err = txClient.WaitSync(sCtx, tr.Uid)
		if err != nil {
			return fmt.Errorf("sync with the node %s: %w", txName, err)
		}

		fmt.Printf("#%d: downloading from the node: %s\n", i, rxName)
		dd, err := rxClient.DownloadBytes(ctx, addr)
		if err != nil {
			return fmt.Errorf("download from node %s: %w", rxName, err)
		}

		if !bytes.Equal(data, dd) {
			return fmt.Errorf("downloaded data mismatch")
		}

		fmt.Printf("#%d: downloaded successfully from node: %s\n", i, rxName)
	}

	fmt.Println("smoke test completed successfully")
	return nil
}

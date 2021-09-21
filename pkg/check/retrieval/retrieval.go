package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	GasPrice        string
	PostageAmount   int64
	PostageDepth    uint64
	PostageLabel    string
	PostageWait     time.Duration
	Seed            int64
	UploadNodeCount int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:   1,
		GasPrice:        "",
		PostageAmount:   1,
		PostageDepth:    16,
		PostageLabel:    "test-label",
		PostageWait:     5 * time.Second,
		Seed:            random.Int64(),
		UploadNodeCount: 1,
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

var errRetrieval = errors.New("retrieval")

// Run uploads given chunks on cluster and checks pushsync ability of the cluster
func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}
	sortedNodes := cluster.NodeNames()
	lastNodeName := sortedNodes[len(sortedNodes)-1]
	for i := 0; i < o.UploadNodeCount; i++ {

		nodeName := sortedNodes[i]
		client := clients[nodeName]

		batchID, err := client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", nodeName, err)
		}
		fmt.Printf("node %s: batch id %s\n", nodeName, batchID)
		time.Sleep(o.PostageWait)

		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			t0 := time.Now()
			ref, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)

			c.metrics.UploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			c.metrics.UploadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d0.Seconds())
			c.metrics.UploadTimeHistogram.Observe(d0.Seconds())

			t1 := time.Now()

			data, err := clients[lastNodeName].DownloadChunk(ctx, ref, "")
			if err != nil {
				return fmt.Errorf("node %s: %w", lastNodeName, err)
			}
			d1 := time.Since(t1)

			c.metrics.DownloadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			c.metrics.DownloadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d1.Seconds())
			c.metrics.DownloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(chunk.Data(), data) {
				c.metrics.NotRetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("Node %s. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", nodeName, j, chunk.Size(), len(data), overlays[nodeName].String(), ref.String())
				if bytes.Contains(chunk.Data(), data) {
					fmt.Printf("Downloaded data is subset of the uploaded data\n")
				}
				return errRetrieval
			}

			c.metrics.RetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			fmt.Printf("Node %s. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", nodeName, j, overlays[nodeName].String(), chunk.Address().String())
		}
	}

	return
}

package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	PostageAmount   int64
	PostageDepth    uint64
	PostageLabel    string
	Seed            int64
	UploadNodeCount int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:   1,
		PostageAmount:   1,
		PostageLabel:    "test-label",
		PostageDepth:    16,
		Seed:            random.Int64(),
		UploadNodeCount: 1,
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
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		metrics: newMetrics(),
		logger:  logger,
	}
}

var errRetrieval = errors.New("retrieval")

// Run uploads given chunks on cluster and checks pushsync ability of the cluster
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	nodes := cluster.FullNodeNames()

	for i := 0; i < o.UploadNodeCount; i++ {
		uploadNode := clients[nodes[i]]

		downloadNodeIndex := (i + 1) % len(nodes) // download from the next node
		downloadNode := clients[nodes[downloadNodeIndex]]

		batchID, err := uploadNode.GetOrCreateMutableBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: created batched id %w", uploadNode.Name(), err)
		}
		c.logger.Infof("node %s: created batched id %s", uploadNode.Name(), batchID)

		for j := 0; j < o.ChunksPerNode; j++ {
			// time upload
			t0 := time.Now()

			chunk, err := bee.NewRandomChunk(rnds[i], c.logger)
			if err != nil {
				return fmt.Errorf("node %s: %w", uploadNode.Name(), err)
			}

			addr, err := uploadNode.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
			if err != nil {
				return fmt.Errorf("node %s: %w", uploadNode.Name(), err)
			}
			c.logger.Infof("Uploaded chunk address: %s", addr.String())

			d0 := time.Since(t0)

			c.metrics.UploadedCounter.WithLabelValues(overlays[uploadNode.Name()].String()).Inc()
			c.metrics.UploadTimeGauge.WithLabelValues(overlays[uploadNode.Name()].String(), chunk.Address().String()).Set(d0.Seconds())
			c.metrics.UploadTimeHistogram.Observe(d0.Seconds())

			// time download
			t1 := time.Now()

			downloadData, err := downloadNode.DownloadChunk(ctx, chunk.Address(), "", nil)
			if err != nil {
				return fmt.Errorf("node %s: %w", downloadNode.Name(), err)
			}

			d1 := time.Since(t1)

			c.metrics.DownloadedCounter.WithLabelValues(uploadNode.Name()).Inc()
			c.metrics.DownloadTimeGauge.WithLabelValues(uploadNode.Name(), chunk.Address().String()).Set(d1.Seconds())
			c.metrics.DownloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(chunk.Data(), downloadData) {
				c.metrics.NotRetrievedCounter.WithLabelValues(uploadNode.Name()).Inc()
				c.logger.Errorf("Chunk not retrieved successfully: DownloadNode=%s, ChunkIndex=%d, UploadedSize=%d, DownloadedSize=%d, UploadNode=%s, ChunkAddress=%s", downloadNode.Name(), j, chunk.Size(), len(downloadData), uploadNode.Name(), chunk.Address().String())
				if bytes.Contains(chunk.Data(), downloadData) {
					c.logger.Infof("Downloaded data is subset of the uploaded data")
				}
				return errRetrieval
			}

			c.metrics.RetrievedCounter.WithLabelValues(uploadNode.Name()).Inc()
			c.logger.Infof("Chunk retrieved successfully: DownloadNode=%s, ChunkIndex=%d, DownloadedSize=%d, UploadNode=%s, ChunkAddress=%s", downloadNode.Name(), j, len(downloadData), uploadNode.Name(), chunk.Address().String())
		}
	}

	return
}

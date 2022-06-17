package retrieval

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	test "github.com/ethersphere/beekeeper/pkg/test"
)

// Options represents check options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	GasPrice        string
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
		GasPrice:        "",
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

	caseOpts := test.CaseOptions{
		GasPrice:      o.GasPrice,
		PostageAmount: o.PostageAmount,
		PostageLabel:  o.PostageLabel,
		PostageDepth:  o.PostageDepth,
		Seed:          o.Seed,
	}

	checkCase, err := test.NewCheckCase(ctx, cluster, caseOpts, c.logger)
	if err != nil {
		return err
	}

	lastBee := checkCase.LastBee()

	for i := 0; i < o.UploadNodeCount; i++ {
		uploader, err := checkCase.Bee(i).NewChunkUploader(ctx)
		if err != nil {
			return err
		}

		for j := 0; j < o.ChunksPerNode; j++ {
			// time upload
			t0 := time.Now()

			chunk, err := uploader.UploadRandomChunk()
			if err != nil {
				return err
			}

			d0 := time.Since(t0)

			c.metrics.UploadedCounter.WithLabelValues(uploader.Overlay).Inc()
			c.metrics.UploadTimeGauge.WithLabelValues(uploader.Overlay, chunk.AddrString()).Set(d0.Seconds())
			c.metrics.UploadTimeHistogram.Observe(d0.Seconds())

			// time download
			t1 := time.Now()

			data, err := lastBee.DownloadChunk(ctx, chunk.Addr())
			if err != nil {
				return fmt.Errorf("node %s: %w", lastBee.Name(), err)
			}

			d1 := time.Since(t1)

			c.metrics.DownloadedCounter.WithLabelValues(uploader.Name()).Inc()
			c.metrics.DownloadTimeGauge.WithLabelValues(uploader.Name(), chunk.AddrString()).Set(d1.Seconds())
			c.metrics.DownloadTimeHistogram.Observe(d1.Seconds())

			if !chunk.Equals(data) {
				c.metrics.NotRetrievedCounter.WithLabelValues(uploader.Name()).Inc()
				c.logger.Infof("Node %s. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", lastBee.Name(), j, chunk.Size(), len(data), uploader.Name(), chunk.AddrString())
				if chunk.Contains(data) {
					c.logger.Infof("Downloaded data is subset of the uploaded data\n")
				}
				return errRetrieval
			}

			c.metrics.RetrievedCounter.WithLabelValues(uploader.Name()).Inc()
			c.logger.Infof("Node %s. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", lastBee.Name(), j, uploader.Name(), chunk.AddrString())
		}
	}

	return
}

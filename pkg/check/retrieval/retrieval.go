package retrieval

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	beeV2 "github.com/ethersphere/beekeeper/pkg/check/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents check options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	PostageDepth    uint64
	MetricsPusher   *push.Pusher
	GasPrice        string
	PostageAmount   int64
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
		MetricsPusher:   nil,
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
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

var errRetrieval = errors.New("retrieval")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.MetricsPusher != nil {
		setUpMetrics(o)
	}

	clusterOpts := beeV2.ClusterOptions{
		PostageDepth:  o.PostageDepth,
		GasPrice:      o.GasPrice,
		PostageAmount: o.PostageAmount,
		PostageLabel:  o.PostageLabel,
		PostageWait:   o.PostageWait,
		Seed:          o.Seed,
	}

	clusterV2, err := beeV2.NewClusterV2(ctx, cluster, clusterOpts)
	if err != nil {
		return err
	}

	lastNode := clusterV2.LastNode()

	for i := 0; i < o.UploadNodeCount; i++ {
		uploader, err := clusterV2.Node(i).NewChunkUploader(ctx)
		if err != nil {
			return err
		}

		time.Sleep(o.PostageWait)

		for j := 0; j < o.ChunksPerNode; j++ {
			// time upload
			t0 := time.Now()

			chunk, err := uploader.UploadRandomChunk()

			if err != nil {
				return err
			}

			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(uploader.Overlay).Inc()
			uploadTimeGauge.WithLabelValues(uploader.Overlay, chunk.AddrString()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			// time download
			t1 := time.Now()

			data, err := lastNode.DownloadChunk(ctx, chunk.Addr())

			if err != nil {
				return fmt.Errorf("node %s: %w", lastNode.Name(), err)
			}

			d1 := time.Since(t1)

			downloadedCounter.WithLabelValues(uploader.Name()).Inc()
			downloadTimeGauge.WithLabelValues(uploader.Name(), chunk.AddrString()).Set(d1.Seconds())
			downloadTimeHistogram.Observe(d1.Seconds())

			if !chunk.Equals(data) {
				notRetrievedCounter.WithLabelValues(uploader.Name()).Inc()
				fmt.Printf("Node %s. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", lastNode.Name(), j, chunk.Size(), len(data), uploader.Name(), chunk.AddrString())
				if chunk.Contains(data) {
					fmt.Printf("Downloaded data is subset of the uploaded data\n")
				}
				return errRetrieval
			}

			retrievedCounter.WithLabelValues(uploader.Name()).Inc()
			fmt.Printf("Node %s. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", lastNode.Name(), j, uploader.Name(), chunk.AddrString())

			if o.MetricsPusher != nil {
				if err := o.MetricsPusher.Push(); err != nil {
					fmt.Printf("node %s: %v\n", lastNode.Name(), err)
				}
			}
		}
	}

	return
}

func setUpMetrics(o Options) {
	o.MetricsPusher.Collector(uploadedCounter)
	o.MetricsPusher.Collector(uploadTimeGauge)
	o.MetricsPusher.Collector(uploadTimeHistogram)
	o.MetricsPusher.Collector(downloadedCounter)
	o.MetricsPusher.Collector(downloadTimeGauge)
	o.MetricsPusher.Collector(downloadTimeHistogram)
	o.MetricsPusher.Collector(retrievedCounter)
	o.MetricsPusher.Collector(notRetrievedCounter)
	o.MetricsPusher.Format(expfmt.FmtText)
}

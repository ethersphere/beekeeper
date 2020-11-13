package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents pushsync check options
type Options struct {
	UploadNodeCount int
	ChunksPerNode   int
	Seed            int64
}

var errRetrieval = errors.New("retrieval")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(uploadedCounter)
	pusher.Collector(uploadTimeGauge)
	pusher.Collector(uploadTimeHistogram)
	pusher.Collector(downloadedCounter)
	pusher.Collector(downloadTimeGauge)
	pusher.Collector(downloadTimeHistogram)
	pusher.Collector(retrievedCounter)
	pusher.Collector(notRetrievedCounter)

	pusher.Format(expfmt.FmtText)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			t0 := time.Now()
			addr, err := c.Nodes[i].UploadChunk(ctx, chunk, api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[i].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[i].String(), addr.String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			t1 := time.Now()
			data, err := c.Nodes[c.Size()-1].DownloadBytes(ctx, addr)
			if err != nil {
				return fmt.Errorf("node %d: %w", c.Size()-1, err)
			}
			d1 := time.Since(t1)

			downloadedCounter.WithLabelValues(overlays[i].String()).Inc()
			downloadTimeGauge.WithLabelValues(overlays[i].String(), addr.String()).Set(d1.Seconds())
			downloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(chunk.Data(), data) {
				notRetrievedCounter.WithLabelValues(overlays[i].String()).Inc()
				fmt.Printf("Node %d. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", i, j, chunk.Size(), len(data), overlays[i].String(), addr.String())
				if bytes.Contains(chunk.Data(), data) {
					fmt.Printf("Downloaded data is subset of the uploaded data\n")
				}
				return errRetrieval
			}

			retrievedCounter.WithLabelValues(overlays[i].String()).Inc()
			fmt.Printf("Node %d. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", i, j, overlays[i].String(), addr.String())

			if pushMetrics {
				if err := pusher.Push(); err != nil {
					fmt.Printf("node %d: %s\n", i, err)
				}
			}
		}
	}

	return
}

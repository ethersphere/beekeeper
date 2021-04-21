package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents pushsync check options
type Options struct {
	NodeGroup       string
	UploadNodeCount int
	ChunksPerNode   int
	Seed            int64
}

var errRetrieval = errors.New("retrieval")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c *bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
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

	overlays, err := c.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := c.NodeNames()
	lastNodeName := sortedNodes[len(sortedNodes)-1]
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			t0 := time.Now()
			clients, err := c.NodesClients(ctx)
			if err != nil {
				return err
			}

			client := clients[nodeName]

			ref, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			t1 := time.Now()

			client = clients[lastNodeName]

			data, err := client.DownloadChunk(ctx, ref, "")
			if err != nil {
				return fmt.Errorf("node %s: %w", lastNodeName, err)
			}
			d1 := time.Since(t1)

			downloadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			downloadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d1.Seconds())
			downloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(chunk.Data(), data) {
				notRetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("Node %s. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", nodeName, j, chunk.Size(), len(data), overlays[nodeName].String(), ref.String())
				if bytes.Contains(chunk.Data(), data) {
					fmt.Printf("Downloaded data is subset of the uploaded data\n")
				}
				return errRetrieval
			}

			retrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			fmt.Printf("Node %s. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", nodeName, j, overlays[nodeName].String(), chunk.Address().String())

			if pushMetrics {
				if err := pusher.Push(); err != nil {
					fmt.Printf("node %s: %v\n", nodeName, err)
				}
			}
		}
	}

	return
}

func flattenOverlays(o bee.ClusterOverlays) map[string]swarm.Address {
	res := make(map[string]swarm.Address)
	for _, ngo := range o {
		for n, over := range ngo {
			res[n] = over
		}
	}
	return res
}

package retrieval

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents simulation options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	MetricsPusher   *push.Pusher
	PostageDepth    uint64
	PostageWait     time.Duration
	Seed            int64
	UploadNodeCount int
	UploadDelay     time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:   1,
		MetricsPusher:   nil,
		PostageDepth:    16,
		PostageWait:     5 * time.Second,
		Seed:            random.Int64(),
		UploadNodeCount: 1,
		UploadDelay:     5 * time.Second,
	}
}

// compile simulation whether Upload implements interface
var _ beekeeper.Action = (*Simulation)(nil)

// Simulation instance
type Simulation struct{}

// NewSimulation returns new upload simulation
func NewSimulation() beekeeper.Action {
	return &Simulation{}
}

// Run executes retrieval simulation
func (s *Simulation) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	metrics := newMetrics(cluster.Name(), o.MetricsPusher)

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	// continually upload chunk and download
	for {
		sortedNodes := cluster.NodeNames()
		for i := 0; i < o.UploadNodeCount; i++ {

			nodeName := sortedNodes[i]
			client := clients[nodeName]

			batchID, err := client.GetOrCreateBatch(ctx, o.PostageDepth, o.PostageWait)
			if err != nil {
				fmt.Printf("error: node %s: batch id %v\n", nodeName, err)
				continue
			}
			fmt.Printf("node %s: batch id %s\n", nodeName, batchID)

			for j := 0; j < o.ChunksPerNode; j++ {
				chunk, err := bee.NewRandomChunk(rnds[i])
				if err != nil {
					fmt.Printf("error: node %s: %v\n", nodeName, err)
					continue
				}

				t0 := time.Now()
				ref, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
				if err != nil {
					fmt.Printf("error: node %s: %v\n", nodeName, err)
					continue
				}
				d0 := time.Since(t0)

				metrics.uploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				metrics.uploadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d0.Seconds())
				metrics.uploadTimeHistogram.Observe(d0.Seconds())

				t1 := time.Now()

				// pick a random node to validate that the chunk is retrievable
				downloadNode := sortedNodes[rnds[i].Intn(len(sortedNodes))]

				data, err := clients[downloadNode].DownloadChunk(ctx, ref, "")
				if err != nil {
					fmt.Printf("error: node %s: %v\n", downloadNode, err)
					continue
				}
				d1 := time.Since(t1)

				metrics.downloadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				metrics.downloadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d1.Seconds())
				metrics.downloadTimeHistogram.Observe(d1.Seconds())

				if !bytes.Equal(chunk.Data(), data) {
					metrics.notRetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					fmt.Printf("Node %s. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", nodeName, j, chunk.Size(), len(data), overlays[nodeName].String(), ref.String())
					if bytes.Contains(chunk.Data(), data) {
						fmt.Printf("Downloaded data is subset of the uploaded data\n")
					}
					continue
				}

				metrics.retrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("Node %s. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", nodeName, j, overlays[nodeName].String(), chunk.Address().String())

				if o.MetricsPusher != nil {
					if err := o.MetricsPusher.Push(); err != nil {
						fmt.Printf("node %s: %v\n", nodeName, err)
					}
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(o.UploadDelay):
		}
	}
}

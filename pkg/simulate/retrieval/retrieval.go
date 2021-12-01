package retrieval

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents simulation options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	GasPrice        string
	PostageAmount   int64
	PostageDepth    uint64
	PostageLabel    string
	Seed            int64
	UploadNodeCount int
	UploadDelay     time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:   1,
		GasPrice:        "",
		PostageAmount:   1000,
		PostageDepth:    16,
		PostageLabel:    "test-label",
		Seed:            random.Int64(),
		UploadNodeCount: 1,
		UploadDelay:     5 * time.Second,
	}
}

// compile simulation whether Upload implements interface
var _ beekeeper.Action = (*Simulation)(nil)

// Simulation instance
type Simulation struct {
	metrics metrics
}

// NewSimulation returns new upload simulation
func NewSimulation() beekeeper.Action {
	return &Simulation{newMetrics("")}
}

// Run executes retrieval simulation
func (s *Simulation) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
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

	// continually upload chunk and download
	for {
		sortedNodes := cluster.NodeNames()
		for i := 0; i < o.UploadNodeCount; i++ {

			nodeName := sortedNodes[i]
			client := clients[nodeName]

			batchID, err := client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
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

				tag, err := client.CreateTag(ctx)
				if err != nil {
					return fmt.Errorf("create tag on node %s: %w", nodeName, err)
				}

				// upload chunk
				t0 := time.Now()
				ref, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{
					BatchID: batchID,
					Tag:     tag.Uid,
				})
				d0 := time.Since(t0)
				if err != nil {
					s.metrics.NotUploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					fmt.Printf("error: node %s: %v\n", nodeName, err)
					continue
				}
				s.metrics.UploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				s.metrics.UploadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d0.Seconds())
				s.metrics.UploadTimeHistogram.Observe(d0.Seconds())
				fmt.Printf("Chunk %s uploaded successfully to node %s\n", chunk.Address().String(), overlays[nodeName].String())

				// check if chunk is synced
				t1 := time.Now()
				err = client.WaitSync(ctx, tag.Uid)
				d1 := time.Since(t1)
				if err != nil {
					s.metrics.NotSyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					fmt.Printf("sync with node %s: %v\n", nodeName, err)
					continue
				}
				s.metrics.SyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				s.metrics.SyncTagsTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d1.Seconds())
				s.metrics.SyncTagsTimeHistogram.Observe(d1.Seconds())
				fmt.Printf("Chunk %s synced successfully with node %s\n", chunk.Address().String(), nodeName)

				// pick a random node to validate that the chunk is retrievable
				downloadNode := sortedNodes[rnds[i].Intn(len(sortedNodes))]

				// download chunk
				t2 := time.Now()
				data, err := clients[downloadNode].DownloadChunk(ctx, ref, "")
				d2 := time.Since(t2)
				if err != nil {
					s.metrics.NotDownloadedCounter.WithLabelValues(overlays[downloadNode].String()).Inc()
					fmt.Printf("error: node %s: %v\n", downloadNode, err)
					continue
				}
				s.metrics.DownloadedCounter.WithLabelValues(overlays[downloadNode].String()).Inc()
				s.metrics.DownloadTimeGauge.WithLabelValues(overlays[downloadNode].String(), ref.String()).Set(d2.Seconds())
				s.metrics.DownloadTimeHistogram.Observe(d2.Seconds())

				// validate that chunk is retrieved correctly
				if !bytes.Equal(chunk.Data(), data) {
					s.metrics.NotRetrievedCounter.WithLabelValues(overlays[downloadNode].String()).Inc()
					fmt.Printf("Node %s. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", downloadNode, j, chunk.Size(), len(data), overlays[downloadNode].String(), ref.String())
					if bytes.Contains(chunk.Data(), data) {
						fmt.Printf("Downloaded data is subset of the uploaded data\n")
					}
					continue
				}
				s.metrics.RetrievedCounter.WithLabelValues(overlays[downloadNode].String()).Inc()
				fmt.Printf("Chunk %s retrieved successfully from node %s\n", chunk.Address().String(), overlays[downloadNode].String())
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(o.UploadDelay):
		}
	}
}

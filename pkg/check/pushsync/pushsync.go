package pushsync

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents pushsync check options
type Options struct {
	UploadNodeCount int
	ChunksPerNode   int
	FilesPerNode    int
	FileSize        int64
	Retries         int
	RetryDelay      time.Duration
	Seed            int64
	PostageAmount   int64
	PostageWait     time.Duration
	PostageDepth    uint64
}

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c *bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) error {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("seed: %d\n", o.Seed)

	pusher.Collector(uploadedCounter)
	pusher.Collector(uploadTimeGauge)
	pusher.Collector(uploadTimeHistogram)
	pusher.Collector(syncedCounter)
	pusher.Collector(notSyncedCounter)

	pusher.Format(expfmt.FmtText)

	overlays, err := c.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	clients, err := c.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := c.NodeNames()
	for i := 0; i < o.UploadNodeCount; i++ {

		nodeName := sortedNodes[i]
		client := clients[nodeName]

		batchID, err := client.GetOrCreateBatch(ctx, o.PostageDepth, o.PostageWait)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", nodeName, err)
		}
		fmt.Printf("node %s: batch id %s\n", nodeName, batchID)

		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			t0 := time.Now()
			addr, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false, BatchID: batchID})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)
			fmt.Printf("uploaded chunk %s to node %s\n", addr.String(), nodeName)

			uploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[nodeName].String(), addr.String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			fmt.Printf("closest node %s overlay %s\n", closestName, closestAddress)

			checkRetryCount := 0

			for {
				checkRetryCount++
				if checkRetryCount > o.Retries {
					return fmt.Errorf("exceeded number of retries")
				}

				time.Sleep(o.RetryDelay)
				node := clients[closestName]
				synced, err := node.HasChunk(ctx, addr)
				if err != nil {
					return fmt.Errorf("node %s: %w", nodeName, err)
				}
				if !synced {
					notSyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					fmt.Printf("node %s overlay %s chunk %s not found on the closest node. retrying...\n", closestName, overlays[closestName], addr.String())
					continue
				}

				syncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("node %s overlay %s chunk %s found on the closest node.\n", closestName, overlays[closestName], addr.String())

				// check succeeded
				break
			}

			if pushMetrics {
				if err := pusher.Push(); err != nil {
					return fmt.Errorf("node %s: %v", nodeName, err)
				}
			}
		}
	}

	return nil
}

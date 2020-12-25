package pushsync

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents pushsync check options
type Options struct {
	NodeGroup       string
	UploadNodeCount int
	ChunksPerNode   int
	FilesPerNode    int
	FileSize        int64
	Retries         int
	RetryDelay      time.Duration
	Seed            int64
}

var errPushSync = errors.New("push sync")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c *bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(uploadedCounter)
	pusher.Collector(uploadTimeGauge)
	pusher.Collector(uploadTimeHistogram)
	pusher.Collector(syncedCounter)
	pusher.Collector(notSyncedCounter)

	pusher.Format(expfmt.FmtText)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			t0 := time.Now()
			addr, err := chunk.Address(), ng.NodeClient(nodeName).UploadChunk(ctx, &chunk, api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[nodeName].String(), addr.String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			checkRetryCount := 0

			for {
				checkRetryCount++
				if checkRetryCount > o.Retries {
					return fmt.Errorf("exceeded number of retires: %w", errPushSync)
				}

				time.Sleep(o.RetryDelay)
				synced, err := ng.NodeClient(closestName).HasChunk(ctx, addr)
				if err != nil {
					return fmt.Errorf("node %s: %w", nodeName, err)
				}
				if !synced {
					notSyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					fmt.Printf("Node %s. Chunk %d not found on the closest node. Node: %s Chunk: %s Closest node %s: %s\n", nodeName, j, overlays[nodeName].String(), addr.String(), closestName, closestAddress.String())
					continue
				}

				syncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("Node %s. Chunk %d found on the closest node. Node: %s Chunk: %s Closest node %s: %s\n", nodeName, j, overlays[nodeName].String(), addr.String(), closestName, closestAddress.String())

				// check succeeded
				break
			}

			if pushMetrics {
				if err := pusher.Push(); err != nil {
					fmt.Printf("node %s: %s\n", nodeName, err)
				}
			}
		}
	}

	return
}

// CheckConcurrent uploads given chunks concurrently on cluster and checks pushsync ability of the cluster
func CheckConcurrent(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]

		var chunkResults []chunkStreamMsg
		for m := range chunkStream(ctx, ng.NodeClient(nodeName), rnds[i], o.ChunksPerNode) {
			chunkResults = append(chunkResults, m)
		}
		for j, c := range chunkResults {
			fmt.Println(i, j, c.Index, c.Chunk.Size(), c.Error)
		}
	}

	return
}

// CheckChunks uploads given chunks on cluster and checks pushsync ability of the cluster
func CheckChunks(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			if err := ng.NodeClient(nodeName).UploadChunk(ctx, &chunk, api.UploadOptions{Pin: false}); err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			time.Sleep(1 * time.Second)
			synced, err := ng.NodeClient(closestName).HasChunk(ctx, chunk.Address())
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			if !synced {
				fmt.Printf("Node %s. Chunk %d not found on the closest node. Node: %s Chunk: %s Closest: %s\n", nodeName, j, overlays[nodeName].String(), chunk.Address().String(), closestAddress.String())
				return errPushSync
			}

			fmt.Printf("Node %s. Chunk %d found on the closest node. Node: %s Chunk: %s Closest: %s\n", nodeName, j, overlays[nodeName].String(), chunk.Address().String(), closestAddress.String())
		}
	}

	return
}

// CheckFiles uploads given files on cluster and verifies expected tag state
func CheckFiles(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.FilesPerNode; j++ {
			rnd := rnds[i]
			fileSize := o.FileSize + int64(j)
			file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d-%d", "file", i, j), fileSize)

			tagResponse, err := ng.NodeClient(nodeName).CreateTag(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			tagUID := tagResponse.Uid

			if err := ng.NodeClient(nodeName).UploadFileWithTag(ctx, &file, false, tagUID); err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlays[nodeName].String())

			checkRetryCount := 0

			for {
				checkRetryCount++
				if checkRetryCount > o.Retries {
					return fmt.Errorf("exceeded number of retires: %w", errPushSync)
				}

				time.Sleep(o.RetryDelay)

				afterUploadTagResponse, err := ng.NodeClient(nodeName).GetTag(ctx, tagUID)
				if err != nil {
					return fmt.Errorf("node %s: %w", nodeName, err)
				}

				tagSplitCount := afterUploadTagResponse.Split
				tagSentCount := afterUploadTagResponse.Sent
				tagSeenCount := afterUploadTagResponse.Seen

				diff := tagSplitCount - (tagSentCount + tagSeenCount)

				if diff != 0 {
					fmt.Printf("File %s tag counters do not match (diff: %d): %+v\n", file.Address().String(), diff, afterUploadTagResponse)
					continue
				}

				fmt.Printf("File %s tag counters: %+v\n", file.Address().String(), afterUploadTagResponse)

				// check succeeded
				break
			}

		}
	}

	return
}

type chunkStreamMsg struct {
	Index int
	Chunk bee.Chunk
	Error error
}

func chunkStream(ctx context.Context, node *bee.Client, rnd *rand.Rand, count int) <-chan chunkStreamMsg {
	chunkStream := make(chan chunkStreamMsg)

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(n *bee.Client, i int) {
			defer wg.Done()
			chunk, err := bee.NewRandomChunk(rnd)
			if err != nil {
				chunkStream <- chunkStreamMsg{Index: i, Error: err}
				return
			}

			if err := n.UploadChunk(ctx, &chunk, api.UploadOptions{Pin: false}); err != nil {
				chunkStream <- chunkStreamMsg{Index: i, Error: err}
				return
			}

			chunkStream <- chunkStreamMsg{Index: i, Chunk: chunk}
		}(node, i)
	}

	go func() {
		wg.Wait()
		close(chunkStream)
	}()

	return chunkStream
}

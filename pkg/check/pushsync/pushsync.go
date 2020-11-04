package pushsync

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
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
}

var errPushSync = errors.New("push sync")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(uploadedCounter)
	pusher.Collector(uploadTimeGauge)
	pusher.Collector(uploadTimeHistogram)
	pusher.Collector(syncedCounter)
	pusher.Collector(notSyncedCounter)

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
			addr, err := c.Nodes[i].UploadBytes(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[i].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[i].String(), addr.String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			closest, err := chunk.ClosestNode(overlays)
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			index := findIndex(overlays, closest)

			time.Sleep(1 * time.Second)
			synced, err := c.Nodes[index].HasChunk(ctx, addr)
			if err != nil {
				return fmt.Errorf("node %d: %w", index, err)
			}
			if !synced {
				notSyncedCounter.WithLabelValues(overlays[i].String()).Inc()
				fmt.Printf("Node %d. Chunk %d not found on the closest node. Node: %s Chunk: %s Closest: %s\n", i, j, overlays[i].String(), addr.String(), closest.String())
				return errPushSync
			}

			syncedCounter.WithLabelValues(overlays[i].String()).Inc()
			fmt.Printf("Node %d. Chunk %d found on the closest node. Node: %s Chunk: %s Closest: %s\n", i, j, overlays[i].String(), addr.String(), closest.String())

			if pushMetrics {
				if err := pusher.Push(); err != nil {
					fmt.Printf("node %d: %s\n", i, err)
				}
			}
		}
	}

	return
}

// CheckConcurrent uploads given chunks concurrently on cluster and checks pushsync ability of the cluster
func CheckConcurrent(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	for i := 0; i < o.UploadNodeCount; i++ {
		var chunkResults []chunkStreamMsg
		for m := range chunkStream(ctx, c.Nodes[i], rnds[i], o.ChunksPerNode) {
			chunkResults = append(chunkResults, m)
		}
		for j, c := range chunkResults {
			fmt.Println(i, j, c.Index, c.Chunk.Size(), c.Error)
		}
	}

	return
}

// CheckChunks uploads given chunks on cluster and checks pushsync ability of the cluster
func CheckChunks(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

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

			if err := c.Nodes[i].UploadChunk(ctx, &chunk, api.UploadOptions{Pin: false}); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			closest, err := chunk.ClosestNode(overlays)
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			index := findIndex(overlays, closest)

			time.Sleep(1 * time.Second)
			synced, err := c.Nodes[index].HasChunk(ctx, chunk.Address())
			if err != nil {
				return fmt.Errorf("node %d: %w", index, err)
			}
			if !synced {
				fmt.Printf("Node %d. Chunk %d not found on the closest node. Node: %s Chunk: %s Closest: %s\n", i, j, overlays[i].String(), chunk.Address().String(), closest.String())
				return errPushSync
			}

			fmt.Printf("Node %d. Chunk %d found on the closest node. Node: %s Chunk: %s Closest: %s\n", i, j, overlays[i].String(), chunk.Address().String(), closest.String())
		}
	}

	return
}

// CheckFiles uploads given files on cluster and verifies expected tag state
func CheckFiles(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.FilesPerNode; j++ {
			rnd := rnds[i]
			fileSize := o.FileSize + int64(j)
			file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d-%d", "file", i, j), fileSize)

			tagResponse, err := c.Nodes[i].CreateTag(ctx)
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			tagUID := tagResponse.Uid

			if err := c.Nodes[i].UploadFileWithTag(ctx, &file, false, tagUID); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlays[i].String())

			checkRetryCount := 0

			for {
				checkRetryCount++
				if checkRetryCount > o.Retries {
					return fmt.Errorf("exceeded number of retires: %w", errPushSync)
				}

				time.Sleep(o.RetryDelay)

				afterUploadTagResponse, err := c.Nodes[i].GetTag(ctx, tagUID)
				if err != nil {
					return fmt.Errorf("node %d: %w", i, err)
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

func chunkStream(ctx context.Context, node bee.Node, rnd *rand.Rand, count int) <-chan chunkStreamMsg {
	chunkStream := make(chan chunkStreamMsg)

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(n bee.Node, i int) {
			defer wg.Done()
			chunk, err := bee.NewRandomChunk(rnd)
			if err != nil {
				chunkStream <- chunkStreamMsg{Index: i, Error: err}
				return
			}

			if _, err := n.UploadBytes(ctx, chunk.Data(), api.UploadOptions{Pin: false}); err != nil {
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

// findIndex returns index of a given swarm.Address in a given set of swarm.Addresses, or -1 if not found
func findIndex(overlays []swarm.Address, addr swarm.Address) int {
	for i, a := range overlays {
		if addr.Equal(a) {
			return i
		}
	}
	return -1
}

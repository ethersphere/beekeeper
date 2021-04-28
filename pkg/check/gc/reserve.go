package gc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents gc check options
type Options struct {
	CacheSize     int // size of the node's localstore in chunks
	Seed          int64
	PostageAmount int64
	PostageWait   time.Duration
	ReserveSize   int
}

func CheckReserve(c *bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Println("gc: reserve check")
	fmt.Printf("Seed: %d\n", o.Seed)

	node, err := c.RandomNode(ctx, rnd)
	if err != nil {
		return fmt.Errorf("random node: %w", err)
	}
	fmt.Printf("node %s\n", node.Name())

	const (
		lowAmonut int64 = 1
		higAmount int64 = 3
	)

	client := node.Client()

	addr, err := client.Addresses(ctx)
	if err != nil {
		return err
	}
	overlay := addr.Overlay

	origState, err := client.ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", origState)
	depth := capacityToDepth(origState.Radius, origState.Available)

	// STEP 1: create low value batch that covers the size of the reserve and upload chunk as much as the size of the cache
	batch, err := client.CreatePostageBatch(ctx, lowAmonut, depth, "test-label")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", batch, depth, higAmount)
	time.Sleep(o.PostageWait)

	state, err := node.Client().ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", state)

	lowValueChunks := chunkBatch(rnd, overlay, int64(o.CacheSize), origState.Radius)
	for _, c := range lowValueChunks {
		_, err := node.Client().UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batch})
		if err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	fmt.Printf("Uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueChunks), depth, lowAmonut, origState.Radius)

	// upload higher radius chunks that should not be garbage collected
	higherRadius := origState.Radius + 1
	higherRadiusChunkCount := int64(float64(o.CacheSize) * 0.1)
	lowValueHigherRadiusChunks := chunkBatch(rnd, overlay, higherRadiusChunkCount, higherRadius)
	for _, c := range lowValueHigherRadiusChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batch}); err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	fmt.Printf("Uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueHigherRadiusChunks), depth, lowAmonut, higherRadius)

	// STEP 2: create high value batch thats covers the size of the reserve which should trigger the garbage collection of the low value batch
	highValueBatch, err := client.CreatePostageBatch(ctx, higAmount, depth, "test-label")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", highValueBatch, depth, higAmount)
	time.Sleep(o.PostageWait)

	state, err = node.Client().ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", state)

	hasCount := 0
	for _, c := range lowValueChunks {
		has, err := client.HasChunk(ctx, c.Address())
		if err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
		if has {
			hasCount++
		}
	}

	fmt.Printf("retrieved low value chunks: %d, gc'd count: %d\n", hasCount, len(lowValueChunks)-hasCount)

	// a 10% cache garbage collection is expected
	if hasCount != int(float64(len(lowValueChunks))*0.9) {
		return errors.New("lowValueChunks gc count  error")
	}

	hasCount = 0
	for _, c := range lowValueHigherRadiusChunks {
		has, err := client.HasChunk(ctx, c.Address())
		if err != nil {
			return fmt.Errorf("low value higher radius chunk: %w", err)
		}
		if has {
			hasCount++
		}
	}

	fmt.Printf("retrieved low value high radius chunks: %d, gc'd count: %d\n", hasCount, len(lowValueHigherRadiusChunks)-hasCount)

	if len(lowValueHigherRadiusChunks) != hasCount {
		return fmt.Errorf("low value higher radius chunks were gc'd. Retrieved: %d, gc'd count: %d", hasCount, len(lowValueHigherRadiusChunks)-hasCount)
	}

	// STEP 3: Upload chunks with high value batch, then create a low value batch, and confirm no chunks were garbage collected
	highValueChunks := chunkBatch(rnd, overlay, int64(o.CacheSize), state.Radius)
	for _, c := range highValueChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: highValueBatch}); err != nil {
			return fmt.Errorf("high value chunks: %w", err)
		}
	}
	fmt.Printf("Uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(highValueChunks), depth, higAmount, state.Radius)

	batch, err = client.CreatePostageBatch(ctx, lowAmonut, depth-1, "test-label")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", batch, depth, higAmount)
	time.Sleep(o.PostageWait)

	state, err = client.ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", state)

	hasCount = 0
	for _, c := range highValueChunks {
		has, err := client.HasChunk(ctx, c.Address())
		if err != nil {
			return fmt.Errorf("high value chunk: %w", err)
		}
		if has {
			hasCount++
		}
	}

	fmt.Printf("retrieved high value chunks: %d, gc'd count: %d\n", hasCount, len(highValueChunks)-hasCount)

	if len(highValueChunks) != hasCount {
		return fmt.Errorf("high value chunks were gc'd. Retrieved: %d,  gc'd count: %d", hasCount, len(highValueChunks)-hasCount)
	}

	return nil
}

func capacityToDepth(radius uint8, available int64) uint64 {
	depth := int(math.Log2(float64(available) * math.Exp2(float64(radius))))
	if depth < bee.MinimumBatchDepth {
		depth = bee.MinimumBatchDepth
	}

	return uint64(depth) + 1
}

func chunkBatch(rnd *rand.Rand, target swarm.Address, count int64, po uint8) []swarm.Chunk {

	ret := make([]swarm.Chunk, 0, count)

	for i := int64(0); i < count; i++ {
		ret = append(ret, bee.GenerateRandomChunkAt(rnd, target, po))
	}

	return ret
}

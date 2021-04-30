package gc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

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
		loAmount = 1
		hiAmount = 3
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
	batchID, err := client.CreatePostageBatch(ctx, loAmount, depth, "test-label")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", batchID, depth, hiAmount)
	time.Sleep(o.PostageWait)

	state, err := node.Client().ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", state)

	pinnedChunk := bee.GenerateRandomChunkAt(rnd, overlay, 0)
	_, err = client.UploadChunk(ctx, pinnedChunk.Data(), api.UploadOptions{Pin: true, BatchID: batchID})
	if err != nil {
		return fmt.Errorf("unable to upload chunk: %w", err)
	}
	fmt.Printf("uploaded pinned chunk %q\n", pinnedChunk.Address())

	lowValueChunks := chunkBatch(rnd, overlay, o.CacheSize, origState.Radius)
	for _, c := range lowValueChunks {
		_, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID})
		if err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueChunks), depth, loAmount, origState.Radius)

	// upload higher radius chunks that should not be garbage collected
	higherRadius := origState.Radius + 1
	higherRadiusChunkCount := int(float64(o.CacheSize) * 0.1)
	lowValueHigherRadiusChunks := chunkBatch(rnd, overlay, higherRadiusChunkCount, higherRadius)
	for _, c := range lowValueHigherRadiusChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID}); err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueHigherRadiusChunks), depth, loAmount, higherRadius)

	// STEP 2: create high value batch that covers the size of the reserve which should trigger the garbage collection of the low value batch
	highValueBatch, err := client.CreatePostageBatch(ctx, hiAmount, depth, "test-label")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", highValueBatch, depth, hiAmount)
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

	has, err := client.HasChunk(ctx, pinnedChunk.Address())
	if err != nil {
		return fmt.Errorf("unable to check pinned chunk %q: %w", pinnedChunk.Address(), err)
	}
	if !has {
		return fmt.Errorf("expected node pin for uploaded chunk %q", pinnedChunk.Address())
	}

	// STEP 3: Upload chunks with high value batch, then create a low value batch, and confirm no chunks were garbage collected
	highValueChunks := chunkBatch(rnd, overlay, o.CacheSize, state.Radius)
	for _, c := range highValueChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: highValueBatch}); err != nil {
			return fmt.Errorf("high value chunks: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(highValueChunks), depth, hiAmount, state.Radius)

	batchID, err = client.CreatePostageBatch(ctx, loAmount, depth-1, "test-label")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", batchID, depth, hiAmount)
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

	pinned, err := client.GetPins(ctx)
	if err != nil {
		return fmt.Errorf("unable to get pins: %w", err)
	}
	if len(pinned) != 1 {
		return fmt.Errorf("unexpected pin count %d", len(pinned))
	}
	pinnedRef := pinned[0]

	if !pinnedChunk.Address().Equal(pinnedRef) {
		return fmt.Errorf("chunk %q is not pinned", pinnedChunk.Address())
	}

	if have, err := client.GetPinnedRootHash(ctx, pinnedRef); err != nil {
		return fmt.Errorf("unable to get pinned root hash: %w", err)
	} else if !have.Equal(pinnedRef) {
		return fmt.Errorf("address mismatch: have %q; want %q", have, pinnedRef)
	}

	if err := client.UnpinRootHash(ctx, pinnedRef); err != nil {
		return fmt.Errorf("cannot unpin chunk: %w", err)
	}

	if have, err := client.GetPinnedRootHash(ctx, pinnedRef); err != nil {
		return fmt.Errorf("unable to get pinned root hash: %w", err)
	} else if !have.Equal(swarm.ZeroAddress) {
		return fmt.Errorf("address mismatch: have %q; want none", have)
	}

	pinned, err = client.GetPins(ctx)
	if err != nil {
		return fmt.Errorf("unable to get pins: %w", err)
	}
	if len(pinned) > 0 {
		return errors.New("pin count is greater than zero")
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

func chunkBatch(rnd *rand.Rand, target swarm.Address, count int, po uint8) []swarm.Chunk {
	chunks := make([]swarm.Chunk, count)
	for i := range chunks {
		chunks[i] = bee.GenerateRandomChunkAt(rnd, target, po)
	}
	return chunks
}

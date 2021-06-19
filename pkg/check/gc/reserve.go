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
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents gc check options
type Options struct {
	CacheSize    int // size of the node's localstore in chunks
	GasPrice     string
	PostageLabel string
	PostageWait  time.Duration
	ReserveSize  int
	Seed         int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		CacheSize:    1000,
		GasPrice:     "",
		PostageLabel: "test-label",
		PostageWait:  5 * time.Second,
		ReserveSize:  1024,
		Seed:         0,
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

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(o.Seed)
	fmt.Println("gc: reserve check")
	fmt.Printf("Seed: %d\n", o.Seed)

	node, err := cluster.RandomNode(ctx, rnd)
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
	batchID, err := client.CreatePostageBatch(ctx, loAmount, depth, o.GasPrice, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", batchID, depth, loAmount)
	time.Sleep(o.PostageWait)

	state, err := node.Client().ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", state)

	// since CacheSize is the same as the reserve size in this test setup,
	// the 64 chunks would fill the reserve up so that 7 chunks are moved
	// from the reserve to the cache. we still need to insert another (64-7) chunks
	lowValueChunks := chunkBatch(rnd, overlay, 10, origState.Radius)
	for _, c := range lowValueChunks {
		_, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID})
		if err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	lowValueChunks2 := chunkBatch(rnd, overlay, 64-10, origState.Radius+1)
	for _, c := range lowValueChunks2 {
		_, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID})
		if err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	lowValueChunks = append(lowValueChunks, lowValueChunks2...)

	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueChunks), depth, loAmount, origState.Radius)

	// now buy another batch so that the batchstore picks up the batch and
	// lines up the appropriate events in sequence in the FIFO queue for
	// eviction

	// STEP 2: create high value batch that covers the size of the reserve which should trigger the garbage collection of the low value batch
	highValueBatch, err := client.CreatePostageBatch(ctx, hiAmount, depth, o.GasPrice, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", highValueBatch, depth, hiAmount)
	time.Sleep(o.PostageWait)

	// upload higher radius chunks that should not be garbage collected
	// but the lower PO chunks should get GCd since eviction would be called on them
	higherRadius := origState.Radius + 2

	// upload half the CacheSize again so that reserve eviction kicks in again
	// and this time also gc kicks in and evicts 10% of the cache
	higherRadiusChunkCount := 64 - 7
	lowValueHigherRadiusChunks := chunkBatch(rnd, overlay, higherRadiusChunkCount, higherRadius)
	for _, c := range lowValueHigherRadiusChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID}); err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueHigherRadiusChunks), depth, loAmount, higherRadius)

	// allow time to sleep so that chunks can get synced and then GCd
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
	highValueChunks := chunkBatch(rnd, overlay, int(float64(o.CacheSize)*0.5), state.Radius)
	for _, c := range highValueChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: highValueBatch}); err != nil {
			return fmt.Errorf("high value chunks: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(highValueChunks), depth, hiAmount, state.Radius)

	batchID, err = client.CreatePostageBatch(ctx, loAmount, depth-1, o.GasPrice, o.PostageLabel)
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

	return nil
}

func capacityToDepth(radius uint8, available int64) uint64 {
	depth := int(math.Log2(float64(available)))
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

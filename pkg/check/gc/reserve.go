package gc

import (
	"context"
	"errors"
	"fmt"
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
	ReserveSize  int
	Seed         int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		CacheSize:    1000,
		GasPrice:     "500000000000",
		PostageLabel: "test-label",
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

/*
*********************************************************
**                       WARNING!!!                    **
*********************************************************

This test depends on a very particular test setup.
If you modify the test, you must make sure that
the setup is still correct. This test is meant to
run ONLY on the CI since the setup is guaranteed
by diff patches that are applied on the appropriate
source files in the bee repo before the test is run.

- These patches are visible under .github/patches in the bee
	repo.
- The patching sequence is visible in the beekeeper github
	workflow under .github/workflows/beekeeper.yaml also in the
	bee repo.

*********************************************************
**                        SETUP											   **
*********************************************************

-	Cluster must be fresh (i.e. no other previous transactions
	made on the underlying eth backend before the cluster is
	brought up)
-	Initial Radius(Default Depth) = 2
-	Bucket Depth = 2
-	Reserve Capacity = 16 chunks
-	Cache Capacity = 10 chunks
- Batch Amount Per Chunk (cheap)			= 1 PLUR
- Batch Amount Per Chunk (expensive)	= 3 PLUR
- Batch Depth = 8

A little bit about how the numbers make sense:
- Batch Depth = 8 means the batch has 2^8 (256) chunks capacity
-	Since reserve capacity is 16 and batch capacity is 256, we
	divide 256/16 to get the number of neighborhoods required to
	store the batch and arrive at 16. Since 16=2^4 we assume that
	the reserve radius after purchasing the batch increases to 4
- Bucket depth is 2, meaning there are 4 buckets in the batch.
	Since we are mining chunks in this test and because bucket
	depth is a small number, we assume that all chunks we are mining
	per batch fall under the same bucket. This is to prevent tests
	flaking due to Payment Required HTTP caused by one of the buckets
	filling up.

 1 chunk pinned at po 0
 7+3 chunks at po(originalInitialRadius) 10
 evicts 7 chunks to the cache
 reserve
*/

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
	fmt.Printf("chosen node %s\n", node.Name())

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

	const (
		cheapBatchAmount     = 1
		expensiveBatchAmount = 3
		batchDepth           = uint64(8)
		initialRadius        = 2
		higherRadius         = initialRadius + 1
		radius1              = 4 // radius expected after purchase of 1st batch
	)

	var (
		pinnedChunk                = bee.GenerateRandomChunkAt(rnd, overlay, 0)
		lowValueChunks             = chunkBatch(rnd, overlay, 10, initialRadius)
		lowValueHigherRadiusChunks = chunkBatch(rnd, overlay, 10, higherRadius)
	)

	batchID, err := client.CreatePostageBatch(ctx, cheapBatchAmount, batchDepth, o.GasPrice, o.PostageLabel, true)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}

	_, err = client.UploadChunk(ctx, pinnedChunk.Data(), api.UploadOptions{Pin: true, BatchID: batchID})
	if err != nil {
		return fmt.Errorf("unable to upload chunk: %w", err)
	}
	fmt.Printf("uploaded pinned chunk %q\n", pinnedChunk.Address())

	for _, c := range lowValueChunks {
		_, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID})
		if err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}

	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueChunks), batchDepth, cheapBatchAmount, origState.Radius)

	highValueBatch, err := client.CreatePostageBatch(ctx, expensiveBatchAmount, batchDepth, o.GasPrice, o.PostageLabel, true)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}

	for _, c := range lowValueHigherRadiusChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID}); err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueHigherRadiusChunks), batchDepth, cheapBatchAmount, higherRadius)

	// allow time to sleep so that chunks can get synced and then GCd
	time.Sleep(5 * time.Second)

	state, err := node.Client().ReserveState(ctx)
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

	lowValueChunksLen := len(lowValueChunks)
	wantCount := int(float64(lowValueChunksLen) * 0.9) // A 10% cache garbage collection is expected.
	fmt.Printf("retrieved low value chunks: %d, gc'd count: %d\n", hasCount, lowValueChunksLen-hasCount)

	if hasCount != wantCount {
		return fmt.Errorf("lowValueChunks gc count: has %d; want %d\n", hasCount, wantCount)
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

	highValueChunks := chunkBatch(rnd, overlay, 5, state.Radius)
	for _, c := range highValueChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: highValueBatch}); err != nil {
			return fmt.Errorf("high value chunks: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(highValueChunks), batchDepth, expensiveBatchAmount, state.Radius)

	batchID, err = client.CreatePostageBatch(ctx, cheapBatchAmount, batchDepth-1, o.GasPrice, o.PostageLabel, true)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}

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

func chunkBatch(rnd *rand.Rand, target swarm.Address, count int, po uint8) []swarm.Chunk {
	chunks := make([]swarm.Chunk, count)
	for i := range chunks {
		chunks[i] = bee.GenerateRandomChunkAt(rnd, target, po)
	}
	return chunks
}

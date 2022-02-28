package gc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
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
		GasPrice:     "",
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

- Cluster must be fresh (i.e. no other previous transactions
  made on the underlying eth backend before the cluster is
  brought up)
- Initial Radius(Default Depth) = 0
- Bucket Depth = 2
- Reserve Capacity = 16 chunks
- Cache Capacity = 10 chunks
- Batch Amount Per Chunk (cheap)			= 1 PLUR
- Batch Amount Per Chunk (expensive)	= 3 PLUR
- Batch Depth = 8

A little bit about how the numbers make sense:
- Batch Depth = 8 means the batch has 2^8 (256) chunks capacity
- Since reserve capacity is 16 and batch capacity is 256, we
  divide 256/16 to get the number of neighborhoods required to
  store the batch and arrive at 16. Since 16=2^4 we assume that
  the reserve radius after purchasing the batch increases to 4
- Bucket depth is 2, meaning there are 4 buckets in the batch.
  Since we are mining chunks in this test and because bucket
  depth is a small number, we assume that all chunks we are mining
  per batch fall under the same bucket. This is to prevent tests
  flaking due to Payment Required HTTP caused by one of the buckets
  filling up.

*********************************************************
**                      SCENARIO										   **
*********************************************************

- Buy an initial batch with depth 8 and amount 1 PLUR per
  chunk. This makes the initial radius go from 0 to 4.
- Upload 1 pinned chunk at bin 0 to the node.
- Upload 10 chunks at the initial radius PO. These 10
  chunks will later be evicted from the reserve to the
  cache. The number 10 is selected since we evict from the
  reserve in PO quantiles and 10 is exactly the size of the
  cache. This means that when these chunks are evicted to
  the cache, cache eviction is immediately triggered, causing
  10% of them (1) to be evicted.
- Upload another 10 chunks, this will result in the reserve
  evicting the first batch of chunks with the first PO in line.
  these are the 10 chunks which were uploaded in the previous
  step.

  NOTE: normally the reserve eviction will try to evict
  from itself either half of the cache size or ten percent of
  the reserve size. We are cheating here by evicting the whole
  cache size in one go due to the fact that all chunks in the
  first PO add up the to entire size of the cache. The reserve
  eviction evicts WHOLE PO quantiles and will not stop in the
  middle of an eviction just because the target was reached.
  It will check whether the target was reached only after a
  certain PO has been kicked out of the reserve.

- Check that ten percent of the first ten chunks have indeed
  been evicted.
  At this stage the cache size is 9 and reserve size 10
- Upload another 5 chunks to the reserve. This should NOT kick
  in the reserve eviction (reserve size 15), and the previous
  chunks in addition to the new ones should remain.
- Check that pinned chunk still persists and perform sanity
  checks on the pinning API.
*/

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {

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
	fmt.Printf("chosen node: %s\n", node.Name())

	client := node.Client()

	addr, err := client.Addresses(ctx)
	if err != nil {
		return err
	}
	overlay := addr.Overlay

	const (
		cheapBatchAmount     = 1
		expensiveBatchAmount = 3
		batchDepth           = uint64(8) // the depth for the batches that we buy

		radiusAfterSecondBatch = 5
	)

	var (
		pinnedChunk                = bee.GenerateRandomChunkAt(rnd, overlay, 0)
		lowValueChunks             = bee.GenerateNRandomChunksAt(rnd, overlay, 10, radiusAfterSecondBatch-1)
		lowValueHigherRadiusChunks = bee.GenerateNRandomChunksAt(rnd, overlay, 10, radiusAfterSecondBatch)
	)

	origState, err := client.ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}

	// do some sanity checks to assure test setup is correct
	if origState.Radius != 0 {
		return fmt.Errorf("wrong initial radius, got %d want %d", origState.Radius, 0)
	}
	if origState.StorageRadius != 0 {
		return fmt.Errorf("wrong initial storage radius, got %d want %d", origState.StorageRadius, 0)
	}
	if origState.Available != 16 {
		return fmt.Errorf("wrong initial storage radius, got %d want %d", origState.Available, 16)
	}

	batchID, err := client.CreatePostageBatch(ctx, cheapBatchAmount, batchDepth, o.GasPrice, o.PostageLabel, true)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	// radius is 4

	_, err = client.UploadChunk(ctx, pinnedChunk.Data(), api.UploadOptions{Pin: true, BatchID: batchID, Deferred: true})
	if err != nil {
		return fmt.Errorf("unable to upload chunk: %w", err)
	}
	fmt.Printf("uploaded pinned chunk %q\n", pinnedChunk.Address())

	for _, c := range lowValueChunks {
		_, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID, Deferred: true})
		if err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}

	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius 4\n", len(lowValueChunks), batchDepth, cheapBatchAmount)

	highValueBatch, err := client.CreatePostageBatch(ctx, expensiveBatchAmount, batchDepth, o.GasPrice, o.PostageLabel, true)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	// radius is 5

	for _, c := range lowValueHigherRadiusChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID, Deferred: true}); err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}

	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius 5\n", len(lowValueHigherRadiusChunks), batchDepth, cheapBatchAmount)

	// allow time to sleep so that chunks can get synced and then GCd
	time.Sleep(5 * time.Second)

	state, err := node.Client().ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("Reserve state:", state)

	if state.StorageRadius != state.Radius {
		return fmt.Errorf("storage radius mismatch. got %d want %d", state.StorageRadius, state.Radius)
	}

	_, hasCount, err := client.HasChunks(ctx, bee.AddressOfChunk(lowValueChunks...))
	if err != nil {
		return fmt.Errorf("low value chunk: %w", err)
	}
	fmt.Printf("retrieved low value chunks: %d, gc'd count: %d\n", hasCount, len(lowValueChunks)-hasCount)

	// cache size is 10 and we expect ten percent to be evicted
	if hasCount != 9 {
		return fmt.Errorf("first batch gc count mismatch, has %d; want %d", hasCount, 9)
	}

	_, hasCount, err = client.HasChunks(ctx, bee.AddressOfChunk(lowValueHigherRadiusChunks...))
	if err != nil {
		return fmt.Errorf("low value higher radius chunk: %w", err)
	}

	fmt.Printf("retrieved low value high radius chunks: %d, gc'd count: %d\n", hasCount, len(lowValueHigherRadiusChunks)-hasCount)

	// expect all chunks to be there
	if hasCount != 10 {
		return fmt.Errorf("higher radius chunks gc'd. got %d want %d", hasCount, 10)
	}

	highValueChunks := bee.GenerateNRandomChunksAt(rnd, overlay, 5, state.Radius)
	for _, c := range highValueChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: highValueBatch, Deferred: true}); err != nil {
			return fmt.Errorf("high value chunks: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(highValueChunks), batchDepth, expensiveBatchAmount, state.Radius)

	_, hasCount, err = client.HasChunks(ctx, bee.AddressOfChunk(highValueChunks...))
	if err != nil {
		return fmt.Errorf("high value chunk: %w", err)
	}

	fmt.Printf("retrieved high value chunks: %d, gc'd count: %d\n", hasCount, len(highValueChunks)-hasCount)

	if len(highValueChunks) != hasCount {
		return fmt.Errorf("high value chunks were gc'd. Retrieved: %d, gc'd count: %d", hasCount, len(highValueChunks)-hasCount)
	}

	// local pinning sanity checks

	has, err := client.HasChunk(ctx, pinnedChunk.Address())
	if err != nil {
		return fmt.Errorf("unable to check pinned chunk %q: %w", pinnedChunk.Address(), err)
	}
	if !has {
		return fmt.Errorf("expected pinned chunk %q to persist", pinnedChunk.Address())
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

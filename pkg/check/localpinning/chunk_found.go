package localpinning

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// CheckChunkFound uploads a single chunk to a node, pins it, then uploads a lot of other chunks to see that it still there
func CheckChunkFound(c *bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	pivot := rnd.Intn(c.Size())
	pivotNode := sortedNodes[pivot]
	chunk, err := bee.NewRandomChunk(rnd)
	if err != nil {
		return err
	}

	buffSize := (o.StoreSize / o.StoreSizeDivisor) * swarm.ChunkSize

	client := ng.NodeClient(pivotNode)

	// add some depth buffer
	depth := 3 + bee.EstimatePostageBatchDepth(int64(buffSize*o.StoreSizeDivisor))
	batchID, err := client.CreatePostageBatch(ctx, o.PostageAmount, depth, "test-label")
	if err != nil {
		return fmt.Errorf("node %s: created batched id %w", pivotNode, err)
	}

	fmt.Printf("node %s: created batched id %s\n", pivotNode, batchID)

	time.Sleep(o.PostageWait)

	ref, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: true, BatchID: batchID})
	if err != nil {
		return fmt.Errorf("node %s: %w", pivotNode, err)
	}

	fmt.Printf("uploaded pinned chunk %s to node %s: %s\n", ref.String(), pivotNode, overlays[pivotNode].String())

	b := make([]byte, buffSize)

	for i := 0; i < o.StoreSizeDivisor; i++ {
		_, err = rnd.Read(b)
		if err != nil {
			return fmt.Errorf("rand read: %w", err)
		}
		if _, err := client.UploadBytes(ctx, b, api.UploadOptions{BatchID: batchID}); err != nil {
			return fmt.Errorf("node %s: %w", pivotNode, err)
		}
		fmt.Printf("node %s: uploaded %d bytes.\n", pivotNode, len(b))
	}

	// allow nodes to sync and do some GC
	time.Sleep(5 * time.Second)

	has, err := client.HasChunk(ctx, chunk.Address())
	if err != nil {
		return fmt.Errorf("node has chunk: %w", err)
	}
	if !has {
		return errors.New("pinning node: chunk not found")
	}

	// cleanup
	if err := client.UnpinChunk(ctx, chunk.Address()); err != nil {
		return fmt.Errorf("unpin chunk: %w", err)
	}

	return nil
}

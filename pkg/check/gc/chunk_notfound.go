package gc

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

// Options represents localpinning check options
type Options struct {
	NodeGroup        string
	StoreSize        int // size of the node's localstore in chunks
	StoreSizeDivisor int // divide store size by how much when uploading bytes
	Seed             int64
	Wait             time.Duration
	PostageAmount    int64
	PostageWait      time.Duration
	ReserveSize      int
}

// CheckChunkNotFound uploads a single chunk to a node, then uploads a lot of other chunks to see that it has been purged with gc
func CheckChunkNotFound(c *bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	node, err := c.RandomNode(ctx, rnd)
	if err != nil {
		return fmt.Errorf("random node: %w", err)
	}
	overlay, err := node.Client().Overlay(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get overlay: %w", node.Name(), err)
	}

	chunk, err := bee.NewRandomChunk(rnd)
	if err != nil {
		return err
	}

	buffSize := (o.StoreSize / o.StoreSizeDivisor) * swarm.ChunkSize

	depth := 3 + bee.EstimatePostageBatchDepth(int64(buffSize*o.StoreSizeDivisor))
	batchID, err := node.Client().CreatePostageBatch(ctx, o.PostageAmount, depth, "test-label")
	if err != nil {
		return fmt.Errorf("node %s: created batched id %w", node.Name(), err)
	}

	fmt.Printf("node %s: created batched id %s\n", node.Name(), batchID)

	time.Sleep(o.PostageWait)

	ref, err := node.Client().UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return fmt.Errorf("node %s: %w", node.Name(), err)
	}
	fmt.Printf("uploaded chunk %s (%d bytes) to node %s: %s\n", ref.String(), len(chunk.Data()), node.Name(), overlay.String())

	b := make([]byte, buffSize)

	for i := 0; i <= o.StoreSizeDivisor; i++ {
		_, err := rnd.Read(b)
		if err != nil {
			return fmt.Errorf("rand read: %w", err)
		}
		if _, err := node.Client().UploadBytes(ctx, b, api.UploadOptions{BatchID: batchID}); err != nil {
			return fmt.Errorf("node %s: %w", node.Name(), err)
		}
		fmt.Printf("node %s: uploaded %d bytes.\n", node.Name(), len(b))
	}

	// allow time for syncing and GC
	time.Sleep(o.Wait)

	has, err := node.Client().HasChunk(ctx, ref)
	if err != nil {
		return fmt.Errorf("gc: %w", err)
	}
	if has {
		return errors.New("gc: chunk found after garbage collection")
	}
	return nil
}

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
	StoreSize        int // size of the node's localstore in chunks
	StoreSizeDivisor int // divide store size by how much when uploading bytes
	Seed             int64
}

// CheckChunkNotFound uploads a single chunk to a node, then uploads a lot of other chunks to see that it has been purged with gc
func CheckChunkNotFound(c bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	pivot := rnd.Intn(c.Size())
	chunk, err := bee.NewRandomChunk(rnd)
	if err != nil {
		return err
	}

	if err := c.Nodes[pivot].UploadChunk(ctx, &chunk, api.UploadOptions{Pin: false}); err != nil {
		return fmt.Errorf("node %d: %w", pivot, err)
	}
	fmt.Printf("uploaded chunk %s (%d bytes) to node %d: %s\n", chunk.Address().String(), len(chunk.Data()), pivot, overlays[pivot].String())

	b := make([]byte, (o.StoreSize/o.StoreSizeDivisor)*swarm.ChunkSize)

	for i := 0; i <= o.StoreSizeDivisor; i++ {
		_, err := rnd.Read(b)
		if err != nil {
			return fmt.Errorf("rand read: %w", err)
		}
		if _, err := c.Nodes[pivot].UploadBytes(ctx, b, api.UploadOptions{Pin: false}); err != nil {
			return fmt.Errorf("node %d: %w", pivot, err)
		}
		fmt.Printf("node %d: uploaded %d bytes.\n", pivot, len(b))
	}

	// allow time for syncing and GC
	time.Sleep(5 * time.Second)

	has, err := c.Nodes[pivot].HasChunk(ctx, chunk.Address())
	if err != nil {
		return fmt.Errorf("node has chunk: %w", err)
	}
	if has {
		return errors.New("expected chunk not found")
	}
	return nil
}

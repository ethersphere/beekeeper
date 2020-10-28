package localpinning

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// CheckChunkFound uploads a single chunk to a node, pins it, then uploads a lot of other chunks to see that it still there
func CheckChunkFound(c bee.Cluster, o Options) error {
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

	if err := c.Nodes[pivot].UploadChunk(ctx, chunk, true); err != nil {
		return fmt.Errorf("node %d: %w", pivot, err)
	}
	fmt.Printf("uploaded pinned chunk %s to node %d: %s\n", chunk.Address().String(), pivot, overlays[pivot].String())

	b := make([]byte, o.StoreSize/o.StoreSizeDivisor*swarm.ChunkSize)
	for i := 0; i < o.StoreSizeDivisor; i++ {
		_, err = rand.Read(b)
		if err != nil {
			return fmt.Errorf("rand read: %w", err)
		}
		if _, err := c.Nodes[pivot].UploadBytes(ctx, b, false); err != nil {
			return fmt.Errorf("node %d: %w", pivot, err)
		}
		fmt.Printf("node %d: uploaded %d bytes.\n", pivot, len(b))
	}

	has, err := c.Nodes[pivot].HasChunk(ctx, chunk.Address())
	if err != nil {
		return fmt.Errorf("node has chunk: %w", err)
	}
	if !has {
		return errors.New("pinning node: chunk not found")
	}

	err = c.Nodes[pivot].UnpinChunk(ctx, chunk.Address())
	if err != nil {
		return fmt.Errorf("unpin chunk: %w", err)
	}
	return nil
}

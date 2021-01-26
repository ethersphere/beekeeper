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

	ref, err := ng.NodeClient(pivotNode).UploadChunk(ctx, &chunk, api.UploadOptions{Pin: true})
	if err != nil {
		return fmt.Errorf("node %s: %w", pivotNode, err)
	}

	fmt.Printf("uploaded pinned chunk %s to node %s: %s\n", ref.String(), pivotNode, overlays[pivotNode].String())

	b := make([]byte, (o.StoreSize/o.StoreSizeDivisor)*swarm.ChunkSize)

	for i := 0; i < o.StoreSizeDivisor; i++ {
		_, err = rnd.Read(b)
		if err != nil {
			return fmt.Errorf("rand read: %w", err)
		}
		if _, err := ng.NodeClient(pivotNode).UploadBytes(ctx, b, api.UploadOptions{Pin: false}); err != nil {
			return fmt.Errorf("node %s: %w", pivotNode, err)
		}
		fmt.Printf("node %s: uploaded %d bytes.\n", pivotNode, len(b))
	}

	// allow nodes to sync and do some GC
	time.Sleep(5 * time.Second)

	has, err := ng.NodeClient(pivotNode).HasChunk(ctx, chunk.Address())
	if err != nil {
		return fmt.Errorf("node has chunk: %w", err)
	}
	if !has {
		return errors.New("pinning node: chunk not found")
	}

	// cleanup
	if err := ng.NodeClient(pivotNode).UnpinChunk(ctx, chunk.Address()); err != nil {
		return fmt.Errorf("unpin chunk: %w", err)
	}

	return nil
}

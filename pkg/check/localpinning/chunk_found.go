package localpinning

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// CheckChunkFound uploads a single chunk to a node, pins it, then uploads a lot of other chunks to see that it still there
func CheckChunkFound(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
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

	for i := 0; i < o.LargeFileCount; i++ {
		f := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, pivot), o.LargeFileSize)
		if err := c.Nodes[pivot].UploadFile(ctx, f, false); err != nil {
			return fmt.Errorf("node %d: %w", pivot, err)
		}
		fmt.Printf("node %d: uploaded %d bytes (%d/%d), hash %s\n", pivot, o.LargeFileSize, i+1, o.LargeFileCount, f.Address().String())
	}

	has, err := nodeHasChunk(ctx, &c.Nodes[pivot], chunk.Address())
	if err != nil {
		return fmt.Errorf("node has chunk: %w", err)
	}
	if !has {
		return errors.New("pinning node: chunk not found")
	}
	return nil
}

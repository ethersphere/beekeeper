package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents pushsync check options
type Options struct {
	UploadNodeCount int
	ChunksPerNode   int
	Seed            int64
}

var errRetrieval = errors.New("retrieval")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			if err := c.Nodes[i].UploadBytes(ctx, &chunk); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			data, err := c.Nodes[c.Size()-1].DownloadBytes(ctx, chunk.Address())
			if err != nil {
				return fmt.Errorf("node %d: %w", c.Size()-1, err)
			}

			if !bytes.Equal(chunk.Data(), data) {
				fmt.Printf("Node %d. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", i, j, chunk.Size(), len(data), overlays[i].String(), chunk.Address().String())
				if bytes.Contains(chunk.Data(), data) {
					fmt.Printf("Downloaded data is subset of the uploaded data\n")
				}
				return errRetrieval
			}

			fmt.Printf("Node %d. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", i, j, overlays[i].String(), chunk.Address().String())
		}
	}

	return
}

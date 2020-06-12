package pullsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents pullsync check options
type Options struct {
	UploadNodeCount int
	ChunksPerNode   int
	Seed            int64
}

var errPullSync = errors.New("pull sync")

// Check uploads given chunks on cluster and checks pullsync ability of the cluster
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

			if err := c.Nodes[i].UploadChunk(ctx, &chunk); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			closest, err := chunk.ClosestNode(overlays)
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			index := findIndex(overlays, closest)

			time.Sleep(1 * time.Second)
			synced, err := c.Nodes[index].HasChunk(ctx, chunk.Address())
			if err != nil {
				return fmt.Errorf("node %d: %w", index, err)
			}
			if !synced {
				fmt.Printf("Node %d. Chunk %d not found on the closest node. Node: %s Chunk: %s Closest: %s\n", i, j, overlays[i].String(), chunk.Address().String(), closest.String())
				return errPullSync
			}

			fmt.Printf("Node %d. Chunk %d found on the closest node. Node: %s Chunk: %s Closest: %s\n", i, j, overlays[i].String(), chunk.Address().String(), closest.String())

			crf, err := c.ChunkReplicationFactor(ctx, chunk.Address())
			if err != nil {
				return fmt.Errorf("node %d: %w", index, err)
			}
			fmt.Println("crf", crf)
		}
	}

	return
}

// findIndex returns index of a given swarm.Address in a given set of swarm.Addresses, or -1 if not found
func findIndex(overlays []swarm.Address, addr swarm.Address) int {
	for i, a := range overlays {
		if addr.Equal(a) {
			return i
		}
	}
	return -1
}

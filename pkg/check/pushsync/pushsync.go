package pushsync

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents pushsync check options
type Options struct {
	UploadNodeCount int
	ChunksPerNode   int
	Seed            int64
}

var errPushSync = errors.New("push sync")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnd := rand.New(rand.NewSource(o.Seed))
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnd)
			if err != nil {
				return err
			}

			if err := c.Nodes[i].UploadChunk(ctx, &chunk); err != nil {
				return err
			}

			closest, err := chunk.ClosestNode(overlays)
			if err != nil {
				return err
			}
			index := findIndex(overlays, closest)

			time.Sleep(1 * time.Second)
			synced, err := c.Nodes[index].HasChunk(ctx, chunk)
			if err != nil {
				return err
			}
			if !synced {
				fmt.Printf("Node %d. Chunk %d not found on the closest node. Node: %s Chunk: %s Closest: %s\n", i, j, overlays[i].String(), chunk.Address().String(), closest.String())
				return errPushSync
			}

			fmt.Printf("Node %d. Chunk %d found on the closest node. Node: %s Chunk: %s Closest: %s\n", i, j, overlays[i].String(), chunk.Address().String(), closest.String())
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

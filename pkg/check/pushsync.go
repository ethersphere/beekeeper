package check

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

var errPushSync = errors.New("pushsync")

// PushSync checks cluster's pushsync ability
func PushSync(c bee.Cluster, chunks map[int]map[int]bee.Chunk) (err error) {
	ctx := context.Background()

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	testFailed := false
	for i := 0; i < len(chunks); i++ {
		fmt.Printf("Node %d:\n", i)
		for j := 0; j < len(chunks[i]); j++ {
			// select data
			chunk := chunks[i][j]
			fmt.Printf("Chunk %d size: %d\n", j, chunk.Size())

			// upload chunk
			if err := c.Nodes[i].UploadChunk(ctx, &chunk); err != nil {
				return err
			}
			fmt.Printf("Chunk %d hash: %s\n", j, chunk.Address())

			// find chunk's closest node
			closest, err := chunk.ClosestNode(overlays)
			if err != nil {
				return err
			}
			index := findIndex(overlays, closest)
			fmt.Printf("Chunk %d closest node: %s\n", j, closest)

			time.Sleep(1 * time.Second)
			// check
			hasChunk, err := c.Nodes[index].HasChunk(ctx, chunk)
			if err != nil {
				return err
			}
			if !hasChunk {
				fmt.Printf("Chunk %d not found on closest node\n", j)
				testFailed = true
			}

			fmt.Printf("Chunk %d found on closest node\n", j)
		}
	}

	if testFailed {
		return errPushSync
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

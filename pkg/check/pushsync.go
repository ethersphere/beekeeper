package check

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
)

var errPushSync = errors.New("pushsync")

// PushSync ...
func PushSync(cluster bee.Cluster, chunks map[int]map[int]bee.Chunk) (err error) {
	ctx := context.Background()

	var overlays []swarm.Address
	for _, n := range cluster.Nodes {
		a, err := n.Debug().Node.Addresses(ctx)
		if err != nil {
			return err
		}

		overlays = append(overlays, a.Overlay)
	}

	testFailed := false
	uploadNodes := cluster.Nodes[:len(chunks)]
	for i, n := range uploadNodes {
		fmt.Printf("Node %d:\n", i)
		for j := 0; j < len(chunks[i]); j++ {
			// select data
			chunk := chunks[i][j]
			fmt.Printf("Chunk %d size: %d\n", j, chunk.Size())

			// upload chunk
			chunk.Address, err = n.UploadChunk(ctx, chunk)
			if err != nil {
				return err
			}
			fmt.Printf("Chunk %d hash: %s\n", j, chunk.Address)

			// find chunk's closest node
			closestNode, err := chunk.ClosestNode(overlays)
			if err != nil {
				return err
			}
			closestIndex := findIndex(overlays, closestNode)
			fmt.Printf("Chunk %d closest node: %s\n", j, closestNode)

			time.Sleep(1 * time.Second)
			// check
			resp, err := cluster.Nodes[closestIndex].Debug().Node.HasChunk(ctx, chunk.Address)
			if resp.Message == "OK" {
				fmt.Printf("Chunk %d found on closest node\n", j)
			} else if err == debugapi.ErrNotFound {
				fmt.Printf("Chunk %d not found on closest node\n", j)
				testFailed = true
			} else if err != nil {
				return err
			}
		}
	}

	if testFailed {
		return errPushSync
	}

	return
}

func findIndex(overlays []swarm.Address, addr swarm.Address) int {
	for i, a := range overlays {
		if addr.Equal(a) {
			return i
		}
	}
	return -1
}

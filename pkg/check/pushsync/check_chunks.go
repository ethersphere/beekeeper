package pushsync

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// CheckChunks uploads given chunks on cluster and checks pushsync ability of the cluster
func CheckChunks(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			ref, err := ng.NodeClient(nodeName).UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			time.Sleep(1 * time.Second)
			synced, err := ng.NodeClient(closestName).HasChunk(ctx, ref)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			if !synced {
				fmt.Printf("Node %s. Chunk %d not found on the closest node. Node: %s Chunk: %s Closest: %s\n", nodeName, j, overlays[nodeName].String(), ref.String(), closestAddress.String())
				return errPushSync
			}

			fmt.Printf("Node %s. Chunk %d found on the closest node. Node: %s Chunk: %s Closest: %s\n", nodeName, j, overlays[nodeName].String(), ref.String(), closestAddress.String())
		}
	}

	return
}

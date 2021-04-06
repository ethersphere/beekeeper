package pushsync

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
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
	testCases:
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			uploader := ng.NodeClient(nodeName)

			ref, err := uploader.UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			time.Sleep(o.RetryDelay)
			synced, err := ng.NodeClient(closestName).HasChunk(ctx, ref)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			if !synced {
				fmt.Printf("Node %s. Chunk %d NOT found in the closest node %s Chunk: %s\n", nodeName, j, closestAddress.String(), ref.String())
				return errPushSync
			}

			fmt.Printf("Node %s. Chunk %d found in the closest node %s Chunk: %s\n", nodeName, j, closestAddress.String(), ref.String())

			uploaderAddr, err := uploader.Overlay(ctx)
			if err != nil {
				return err
			}

			skipPeers := []swarm.Address{closestAddress, uploaderAddr}
			// chunk should be replicated at least once either during forwarding or after storing
			for range overlays {
				name, address, err := chunk.ClosestNodeFromMap(overlays, skipPeers...)
				skipPeers = append(skipPeers, address)
				if err != nil {
					continue
				}
				synced, err = ng.NodeClient(name).HasChunk(ctx, ref)
				if err != nil {
					continue
				}
				if synced {
					fmt.Printf("Node %s. Chunk %d was replicated to node %s Chunk: %s\n", name, j, address.String(), ref.String())
					continue testCases
				}
			}

			fmt.Printf("Node %s. Chunk %d NOT replicated to a node\n", nodeName, j)
			return errPushSync
		}
	}

	return
}

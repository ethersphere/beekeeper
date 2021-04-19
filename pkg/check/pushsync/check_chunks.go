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
func CheckChunks(c *bee.Cluster, o Options) error {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("seed: %d\n", o.Seed)

	for _, ng := range c.NodeGroups() {
		overlays, err := ng.Overlays(ctx)
		if err != nil {
			return err
		}

		cc := chunkChecker{
			o:        o,
			ng:       ng,
			overlays: overlays,
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

				fmt.Printf("uploaded chunk %s to node %s\n", ref.String(), nodeName)

				code, err := cc.check(ctx, nodeName, chunk, ref)
				if err != nil {
					return err
				}

				switch code {
				case 1:
					continue //break outer loop
				case 2:
					continue testCases
				default:
					return fmt.Errorf("node %s chunk %s not replicated", nodeName, ref.String())
				}
			}
		}
	}

	return nil
}

type chunkChecker struct {
	o        Options
	ng       *bee.NodeGroup
	overlays bee.NodeGroupOverlays
}

func (c chunkChecker) check(ctx context.Context, nodeName string, chunk bee.Chunk, ref swarm.Address) (int, error) {
	closestName, closestAddress, err := chunk.ClosestNodeFromMap(c.overlays)
	if err != nil {
		return 0, fmt.Errorf("node %s: %w", nodeName, err)
	}
	fmt.Printf("closest node %s overlay %s\n", closestName, closestAddress)

	time.Sleep(c.o.RetryDelay)

	synced, err := c.ng.NodeClient(closestName).HasChunk(ctx, ref)
	if err != nil {
		return 0, fmt.Errorf("node %s: %w", nodeName, err)
	}
	if !synced {
		return 0, fmt.Errorf("node %s chunk %s not found in the closest node %s", nodeName, ref.String(), closestAddress)
	}

	fmt.Printf("node %s chunk %s found in the closest node %s\n", nodeName, ref.String(), closestAddress)

	uploader := c.ng.NodeClient(nodeName)

	uploaderAddr, err := uploader.Overlay(ctx)
	if err != nil {
		return 0, err
	}

	skipPeers := []swarm.Address{closestAddress, uploaderAddr}
	// chunk should be replicated at least once either during forwarding or after storing
	for range c.overlays {
		name, address, err := chunk.ClosestNodeFromMap(c.overlays, skipPeers...)
		skipPeers = append(skipPeers, address)
		if err != nil {
			return 1, nil
		}
		synced, err = c.ng.NodeClient(name).HasChunk(ctx, ref)
		if err != nil {
			return 1, nil
		}
		if synced {
			fmt.Printf("node %s chunk %s was replicated to node %s\n", name, ref.String(), address.String())
			return 2, nil
		}
	}

	return 0, nil
}

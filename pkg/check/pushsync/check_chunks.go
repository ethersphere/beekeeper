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

// checkChunks uploads given chunks on cluster and checks pushsync ability of the cluster
func checkChunks(ctx context.Context, c *bee.Cluster, o Options) error {
	fmt.Println("running pushsync (chunks mode)")
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("seed: %d\n", o.Seed)

	overlays, err := c.FlattenOverlays(ctx)
	if err != nil {
		return err
	}
	clients, err := c.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := c.NodeNames()
	for i := 0; i < o.UploadNodeCount; i++ {

		nodeName := sortedNodes[i]

		uploader := clients[nodeName]

		batchID, err := uploader.GetOrCreateBatch(ctx, o.GasPrice, o.PostageAmount, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", nodeName, err)
		}
		fmt.Printf("node %s: batch id %s\n", nodeName, batchID)
		time.Sleep(o.PostageWait)

	testCases:
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			ref, err := uploader.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			fmt.Printf("uploaded chunk %s to node %s\n", ref.String(), nodeName)

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			fmt.Printf("closest node %s overlay %s\n", closestName, closestAddress)

			time.Sleep(o.RetryDelay)
			synced, err := clients[closestName].HasChunk(ctx, ref)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			if !synced {
				return fmt.Errorf("node %s chunk %s not found in the closest node %s", nodeName, ref.String(), closestAddress)
			}

			fmt.Printf("node %s chunk %s found in the closest node %s\n", nodeName, ref.String(), closestAddress)

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
				synced, err = clients[name].HasChunk(ctx, ref)
				if err != nil {
					continue
				}
				if synced {
					fmt.Printf("node %s chunk %s was replicated to node %s\n", name, ref.String(), address.String())
					continue testCases
				}
			}

			return fmt.Errorf("node %s chunk %s not replicated", nodeName, ref.String())
		}
	}

	return nil
}

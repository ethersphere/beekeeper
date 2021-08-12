package pushsync

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// checkChunks uploads given chunks on cluster and checks pushsync ability of the cluster
func checkLightChunks(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("seed: %d\n", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx, o.ExcludeNodeGroups...)
	if err != nil {
		return err
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	for i, nodeName := range cluster.LightNodeNames() {
		if i >= o.UploadNodeCount {
			break
		}
	testCases:
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			uploader := clients[nodeName]

			batchID, err := uploader.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
			if err != nil {
				return fmt.Errorf("node %s: batch id %w", nodeName, err)
			}
			fmt.Printf("node %s: batch id %s\n", nodeName, batchID)

			var ref swarm.Address

			for i := 0; i < 3; i++ {
				ref, err = uploader.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
				if err == nil {
					break
				}
				time.Sleep(5 * time.Second)
			}

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

			node := clients[closestName]

			synced, err := node.HasChunk(ctx, ref)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			if !synced {
				return fmt.Errorf("node %s chunk %s not found in the closest node %s", nodeName, ref.String(), closestAddress)
			}

			fmt.Printf("node %s chunk %s found in the closest node %s\n", nodeName, ref.String(), closestAddress)

			skipPeers := []swarm.Address{closestAddress}
			// chunk should be replicated at least once either during forwarding or after storing
			for range overlays {
				name, address, err := chunk.ClosestNodeFromMap(overlays, skipPeers...)
				skipPeers = append(skipPeers, address)
				if err != nil {
					continue
				}
				node := clients[name]

				synced, err = node.HasChunk(ctx, ref)
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

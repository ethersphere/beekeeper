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
func checkLightChunks(ctx context.Context, cluster *bee.Cluster, o Options) error {
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("seed: %d\n", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx, "bee", "bootnode")
	if err != nil {
		return err
	}

	lightnodes, err := cluster.NodeGroup("light")
	if err != nil {
		return err
	}

	for i, nodeName := range lightnodes.NodesSorted() {
		if i >= o.UploadNodeCount {
			break
		}
	testCases:
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			uploader, err := lightnodes.NodeClient(nodeName)
			if err != nil {
				return err
			}

			ref, err := uploader.UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
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

			clients, err := cluster.NodesClients(ctx)
			if err != nil {
				return err
			}
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

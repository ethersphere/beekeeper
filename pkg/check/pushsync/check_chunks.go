package pushsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/bee/v2/pkg/topology"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// checkChunks uploads given chunks on cluster and checks pushsync ability of the cluster
func checkChunks(ctx context.Context, c orchestration.Cluster, o Options, l logging.Logger) error {
	l.Info("running pushsync (chunks mode)")
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	l.Infof("seed: %d", o.Seed)

	overlays, err := c.FlattenOverlays(ctx, o.ExcludeNodeGroups...)
	if err != nil {
		return err
	}
	clients, err := c.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := c.FullNodeNames()

	for i := 0; i < o.UploadNodeCount; i++ {

		nodeName := sortedNodes[i]

		uploader := clients[nodeName]

		batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", nodeName, err)
		}
		l.Infof("node %s: batch id %s", nodeName, batchID)

	testCases:
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i], l)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			ref, err := uploader.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			l.Infof("uploaded chunk %s to node %s", ref.String(), nodeName)

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			l.Infof("closest node %s overlay %s", closestName, closestAddress)

			time.Sleep(o.RetryDelay)
			synced, err := clients[closestName].HasChunk(ctx, ref)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			if !synced {
				return fmt.Errorf("node %s chunk %s not found in the closest node %s", nodeName, ref.String(), closestAddress)
			}

			l.Infof("node %s chunk %s found in the closest node %s", nodeName, ref.String(), closestAddress)

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
					if errors.Is(err, topology.ErrNotFound) {
						continue testCases
					}
					continue
				}
				synced, err = clients[name].HasChunk(ctx, ref)
				if err != nil {
					continue
				}
				if synced {
					l.Infof("node %s chunk %s was replicated to node %s", name, ref.String(), address.String())
					continue testCases
				}
			}

			return fmt.Errorf("node %s chunk %s not replicated", nodeName, ref.String())
		}
	}

	return nil
}

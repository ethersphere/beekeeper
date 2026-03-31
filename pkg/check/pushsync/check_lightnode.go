package pushsync

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// checkLightChunks uploads given chunks on cluster and checks pushsync ability of the cluster
func checkLightChunks(ctx context.Context, cluster orchestration.Cluster, o Options, l logging.Logger, m metrics) error {
	l.Info("running pushsync (light-chunks mode)")
	rnd := random.PseudoGenerator(o.Seed)
	l.Infof("seed: %d", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx, o.ExcludeNodeGroups...)
	if err != nil {
		return err
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	lightNodes := cluster.LightNodeNames()

	// prepare postage batches
	for i := range lightNodes {
		nodeName := lightNodes[i]
		batchID, err := clients[nodeName].GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", nodeName, err)
		}
		l.Infof("node %s: batch id %s", nodeName, batchID)
	}

	for i := 0; i < o.UploadNodeCount && i < len(lightNodes); i++ {

		nodeName := lightNodes[i]

		uploader := clients[nodeName]
		batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", nodeName, err)
		}
		l.Infof("node %s: batch id %s", nodeName, batchID)

	testCases:
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnd, l)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			var ref swarm.Address

			t0 := time.Now()
			for range 3 {
				ref, err = uploader.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
				if err == nil {
					break
				}
				time.Sleep(o.RetryDelay)
			}
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)

			l.Infof("uploaded chunk %s to node %s", ref.String(), nodeName)

			m.UploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			m.UploadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d0.Seconds())
			m.UploadTimeHistogram.Observe(d0.Seconds())

			time.Sleep(o.RetryDelay)

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			l.Infof("closest node %s overlay %s", closestName, closestAddress)

			var synced bool
			for range 3 {
				synced, _ = clients[closestName].HasChunk(ctx, ref)
				if synced {
					break
				}
				time.Sleep(o.RetryDelay)
			}
			if !synced {
				m.NotSyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				return fmt.Errorf("node %s chunk %s not found in the closest node %s", nodeName, ref.String(), closestAddress)
			}

			m.SyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			l.Infof("node %s chunk %s found in the closest node %s", nodeName, ref.String(), closestAddress)

			skipPeers := []swarm.Address{closestAddress}
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
					l.Infof("node %s chunk %s was replicated to node %s", name, ref.String(), address.String())
					continue testCases
				}
			}

			return fmt.Errorf("node %s chunk %s not replicated", nodeName, ref.String())
		}
	}

	return nil
}

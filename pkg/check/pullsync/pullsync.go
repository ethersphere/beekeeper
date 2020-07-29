package pullsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents pullsync check options
type Options struct {
	UploadNodeCount            int
	ChunksPerNode              int
	ReplicationFactorThreshold int
	Seed                       int64
}

var errPullSync = errors.New("pull sync")

// Check uploads given chunks on cluster and checks pullsync ability of the cluster
func Check(c bee.Cluster, o Options) (err error) {
	var (
		ctx                    = context.Background()
		rnds                   = random.PseudoGenerators(o.Seed, o.UploadNodeCount)
		totalReplicationFactor float64
	)

	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	topologies, err := c.Topologies(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.ChunksPerNode; j++ {
			var (
				chunk               bee.Chunk
				err                 error
				replicatingNodes    []swarm.Address
				nnRep, peerPoBinRep int
			)

			chunk, err = bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			if err := c.Nodes[i].UploadBytes(ctx, &chunk); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			uploaderToChunkPo := swarm.Proximity(overlays[i].Bytes(), chunk.Address().Bytes())
			fmt.Printf("Uploaded chunk %s\n", chunk.Address())

			// check uploader and non-NN replication
			uploaderTopology := topologies[i]
			for _, bin := range uploaderTopology.Bins {
				for _, peer := range bin.ConnectedPeers {
					peer := peer
					pivotToUploaderPo := swarm.Proximity(peer.Bytes(), overlays[i].Bytes())
					pidx := findIndex(overlays, peer)
					pivotTopology := topologies[pidx]
					pivotDepth := pivotTopology.Depth
					switch pivotPo := swarm.Proximity(chunk.Address().Bytes(), peer.Bytes()); {
					case pivotPo >= pivotDepth:
						// chunk within replicating node depth
						if findIndex(replicatingNodes, peer) == -1 {
							replicatingNodes = append(replicatingNodes, peer)
							nnRep++
						}
					case pivotPo != 0 && uploaderToChunkPo != 0 && pivotPo < pivotDepth && uploaderToChunkPo == pivotToUploaderPo:
						// if the po of the chunk with the uploader == to our po with the uploader, then we need to sync it
						// chunk outside our depth
						if findIndex(replicatingNodes, peer) == -1 {
							replicatingNodes = append(replicatingNodes, peer)
							peerPoBinRep++
						}
					}
				}
			}

			// check closest and NN replication (non-nn replication is not realistic)
			closest, err := chunk.ClosestNode(overlays)
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			index := findIndex(overlays, closest)
			fmt.Printf("Upload node %d. Chunk: %d. Closest: %d %s\n", i, j, index, closest.String())

			topology, err := c.Nodes[index].Topology(ctx)
			if err != nil {
				return fmt.Errorf("node %d: %w", index, err)
			}
			po := swarm.Proximity(chunk.Address().Bytes(), closest.Bytes())
			for _, v := range topology.Bins {
				for _, peer := range v.ConnectedPeers {
					peer := peer
					pivotToClosestPo := swarm.Proximity(peer.Bytes(), closest.Bytes())
					pidx := findIndex(overlays, peer)
					pivotTopology := topologies[pidx]
					pivotDepth := pivotTopology.Depth
					switch pivotPo := swarm.Proximity(chunk.Address().Bytes(), peer.Bytes()); {
					case pivotPo >= pivotDepth:
						// chunk within replicating node depth
						if findIndex(replicatingNodes, peer) == -1 {
							replicatingNodes = append(replicatingNodes, peer)
							nnRep++
						}
					case pivotPo != 0 && pivotPo < pivotDepth && po == pivotToClosestPo:
						// if the po of the chunk with the closest == to our po with the closest, then we need to sync it
						// chunk outside our depth
						// po with chunk must equal po with closest
						if findIndex(replicatingNodes, peer) == -1 {
							replicatingNodes = append(replicatingNodes, peer)
							peerPoBinRep++
						}
					}
				}
			}

			if len(replicatingNodes) == 0 {
				fmt.Printf("Upload node %d. Chunk: %d. Chunk does not have any designated replicators.\n", i, j)
				return errPullSync
			}

			time.Sleep(5 * time.Second)
			fmt.Printf("Chunk should be on %d nodes. %d within depth, %d outside\n", len(replicatingNodes), nnRep, peerPoBinRep)
			for _, n := range replicatingNodes {
				ni := findIndex(overlays, n)

				synced, err := c.Nodes[ni].HasChunk(ctx, chunk.Address())
				if err != nil {
					return fmt.Errorf("node %d: %w", ni, err)
				}
				if !synced {
					return fmt.Errorf("Upload node %d. Chunk %d not found on node. Upload node: %s Chunk: %s Pivot: %s\n", i, j, overlays[i].String(), chunk.Address().String(), n)
				}
			}

			rf, err := c.GlobalReplicationFactor(ctx, chunk.Address())
			if err != nil {
				return fmt.Errorf("replication factor: %w", err)
			}

			if rf < o.ReplicationFactorThreshold {
				fmt.Errorf("chunk %s has low replication factor. got %d want %d", chunk.Address().String(), rf, o.ReplicationFactorThreshold)
			}
			totalReplicationFactor += float64(rf)
			fmt.Printf("Chunk replication factor %d\n", rf)
		}
	}

	totalReplicationFactor = totalReplicationFactor / float64(o.UploadNodeCount*o.ChunksPerNode)
	fmt.Printf("Done with average replication factor: %f\n", totalReplicationFactor)

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

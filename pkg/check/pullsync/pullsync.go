package pullsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents pullsync check options
type Options struct {
	NodeGroup                  string
	UploadNodeCount            int
	ChunksPerNode              int
	ReplicationFactorThreshold int
	Seed                       int64
}

var errPullSync = errors.New("pull sync")

// Check uploads given chunks on cluster and checks pullsync ability of the cluster
func Check(c *bee.DynamicCluster, o Options) (err error) {
	var (
		ctx                    = context.Background()
		rnds                   = random.PseudoGenerators(o.Seed, o.UploadNodeCount)
		totalReplicationFactor float64
	)

	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	topologies, err := ng.Topologies(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.ChunksPerNode; j++ {
			var (
				chunk               bee.Chunk
				err                 error
				nnRep, peerPoBinRep int
			)
			replicatingNodes := make(map[string]swarm.Address)

			chunk, err = bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			addr, err := ng.Node(nodeName).UploadBytes(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			uploaderToChunkPo := swarm.Proximity(overlays[nodeName].Bytes(), addr.Bytes())
			fmt.Printf("Uploaded chunk %s\n", addr.String())

			// check uploader and non-NN replication
			uploaderTopology := topologies[nodeName]
			for _, bin := range uploaderTopology.Bins {
				for _, peer := range bin.ConnectedPeers {
					peer := peer
					pivotToUploaderPo := swarm.Proximity(peer.Bytes(), overlays[nodeName].Bytes())
					pidx := findName(overlays, peer)
					pivotTopology := topologies[pidx]
					pivotDepth := pivotTopology.Depth
					switch pivotPo := int(swarm.Proximity(addr.Bytes(), peer.Bytes())); {
					case pivotPo >= pivotDepth:
						// chunk within replicating node depth
						if len(findName(replicatingNodes, peer)) == 0 {
							replicatingNodes[findName(overlays, peer)] = peer
							nnRep++
						}
					case pivotPo != 0 && uploaderToChunkPo != 0 && pivotPo < pivotDepth && uploaderToChunkPo == pivotToUploaderPo:
						// if the po of the chunk with the uploader == to our po with the uploader, then we need to sync it
						// chunk outside our depth
						if len(findName(replicatingNodes, peer)) == 0 {
							replicatingNodes[findName(overlays, peer)] = peer
							peerPoBinRep++
						}
					}
				}
			}

			// check closest and NN replication (non-nn replication is not realistic)
			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			fmt.Printf("Upload node %s. Chunk: %d. Closest: %s %s\n", nodeName, j, closestName, closestAddress.String())

			topology, err := ng.Node(closestName).Topology(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", closestName, err)
			}
			po := swarm.Proximity(addr.Bytes(), closestAddress.Bytes())
			for _, v := range topology.Bins {
				for _, peer := range v.ConnectedPeers {
					peer := peer
					pivotToClosestPo := swarm.Proximity(peer.Bytes(), closestAddress.Bytes())
					pidx := findName(overlays, peer)
					pivotTopology := topologies[pidx]
					pivotDepth := pivotTopology.Depth
					switch pivotPo := int(swarm.Proximity(addr.Bytes(), peer.Bytes())); {
					case pivotPo >= pivotDepth:
						// chunk within replicating node depth
						if len(findName(replicatingNodes, peer)) == 0 {
							replicatingNodes[findName(overlays, peer)] = peer
							nnRep++
						}
					case pivotPo != 0 && pivotPo < pivotDepth && po == pivotToClosestPo:
						// if the po of the chunk with the closest == to our po with the closest, then we need to sync it
						// chunk outside our depth
						// po with chunk must equal po with closest
						if len(findName(replicatingNodes, peer)) == 0 {
							replicatingNodes[findName(overlays, peer)] = peer
							peerPoBinRep++
						}
					}
				}
			}

			if len(replicatingNodes) == 0 {
				fmt.Printf("Upload node %s. Chunk: %d. Chunk does not have any designated replicators.\n", nodeName, j)
				return errPullSync
			}

			fmt.Printf("Chunk should be on %d nodes. %d within depth, %d outside\n", len(replicatingNodes), nnRep, peerPoBinRep)
			for _, n := range replicatingNodes {
				ni := findName(overlays, n)
				var (
					synced bool
					err    error
				)

				for t := 1; t < 5; t++ {
					time.Sleep(2 * time.Duration(t) * time.Second)
					synced, err = ng.Node(ni).HasChunk(ctx, addr)
					if err != nil {
						return fmt.Errorf("node %s: %w", ni, err)
					}
					if synced {
						break
					}
					fmt.Printf("Upload node %s. Chunk %d not found on node. Upload node: %s Chunk: %s Pivot: %s. Retrying...\n", nodeName, j, overlays[nodeName].String(), addr.String(), n)
				}
				if !synced {
					return fmt.Errorf("Upload node %s. Chunk %d not found on node. Upload node: %s Chunk: %s Pivot: %s", nodeName, j, overlays[nodeName].String(), addr.String(), n)
				}
			}

			rf, err := c.GlobalReplicationFactor(ctx, addr)
			if err != nil {
				return fmt.Errorf("replication factor: %w", err)
			}

			if rf < o.ReplicationFactorThreshold {
				return fmt.Errorf("chunk %s has low replication factor. got %d want %d", addr.String(), rf, o.ReplicationFactorThreshold)
			}
			totalReplicationFactor += float64(rf)
			fmt.Printf("Chunk replication factor %d\n", rf)
		}
	}

	totalReplicationFactor = totalReplicationFactor / float64(o.UploadNodeCount*o.ChunksPerNode)
	fmt.Printf("Done with average replication factor: %f\n", totalReplicationFactor)

	return
}

// findName returns node name of a given swarm.Address in a given set of swarm.Addresses, or "" if not found
func findName(nodes map[string]swarm.Address, addr swarm.Address) string {
	for n, a := range nodes {
		if addr.Equal(a) {
			return n
		}
	}

	return ""
}

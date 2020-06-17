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
	UploadNodeCount int
	ChunksPerNode   int
	Seed            int64
}

var errPullSync = errors.New("pull sync")

// Check uploads given chunks on cluster and checks pullsync ability of the cluster
func Check(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	topologies, err := c.Topologies(ctx)
	if err != nil {
		return err
	}

	// find closest node to chunk
	// go to all nodes which are connected to this node and check their topology
	// if the PO(chunk,pivot) >= depth(pivot) (pivot is the node connected to the closest), then chunk should be synced
	// if the PO(chunk,pivot) < depth(pivot) && PO(chunk,closest) == PO(pivot,closest) (chunk outside depth and equals peerPO bin - should be synced)
	for i := 0; i < o.UploadNodeCount; i++ {
		cpo := 1
		for j := 0; j < o.ChunksPerNode; j++ {
			over := overlays[i]
			var chunk bee.Chunk
			var err error
			start := time.Now()
			for iterate := true; iterate; {
				chunk, err = bee.NewRandomChunk(rnds[i])
				if err != nil {
					return fmt.Errorf("node %d: %w", i, err)
				}

				if ppo := swarm.Proximity(chunk.Address().Bytes(), over.Bytes()); ppo == cpo {
					cpo += 2
					iterate = false
				} else {
					fmt.Printf("mined chunk with different po %d\n", ppo)
				}
			}

			fmt.Printf("took %s to mine chunk", time.Since(start))
			if err := c.Nodes[i].UploadBytes(ctx, &chunk); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

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
			var (
				replicatingNodes    []swarm.Address
				nnRep, peerPoBinRep int
			)

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
						replicatingNodes = append(replicatingNodes, peer)
						fmt.Printf("nn rep. pivotPo %d pivotDepth %d, closestPo %d\n", pivotPo, pivotDepth, po)
						nnRep++
					case pivotPo != 0 && pivotPo < pivotDepth && po == pivotToClosestPo:
						// if the po of the chunk with the closest == to our po with the closest, then we need to sync it
						// chunk outside our depth
						// po with chunk must equal po with closest
						replicatingNodes = append(replicatingNodes, peer)
						fmt.Printf("no nn rep. pivotPo %d pivotDepth %d, pivotToClosestPo %d, closestPo %d\n", pivotPo, pivotDepth, pivotToClosestPo, po)
						peerPoBinRep++
					}
				}
			}

			if len(replicatingNodes) == 0 {
				fmt.Printf("Upload node %d. Chunk: %d. Chunk does not have any designated replicators.", i, j)
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
					fmt.Printf("Upload node %d. Chunk %d not found on node. Upload node: %s Chunk: %s Pivot: %s\n", i, j, overlays[i].String(), chunk.Address().String(), n)
					continue
				}
			}
		}
	}

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

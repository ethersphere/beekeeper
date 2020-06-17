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
	// if the PO(chunk,closest) >= depth(pivot) (pivot is the node connected to the closest), then chunk should be synced
	// if the PO(chunk,closest) < depth(pivot) && PO(chunk,closest) == PO(pivot,closest) (chunk outside depth and equals peerPO bin - should be synced)
	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

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
			var replicatingNodes []swarm.Address

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
					case po < pivotDepth && pivotToClosestPo == pivotPo:
						// chunk outside our depth
						// po with chunk must equal po with closest
						replicatingNodes = append(replicatingNodes, peer)
					}
				}
			}

			if len(replicatingNodes) == 0 {
				fmt.Printf("Upload node %d. Chunk: %d. Chunk does not have any designated replicators. Proximity: %d Depth: %d\n", i, j, po, topology.Depth)
				return errPullSync
			}

			time.Sleep(5 * time.Second)
			for _, n := range replicatingNodes {
				ni := findIndex(overlays, n)

				synced, err := c.Nodes[ni].HasChunk(ctx, chunk.Address())
				if err != nil {
					return fmt.Errorf("node %d: %w", ni, err)
				}
				if !synced {
					fmt.Printf("Upload node %d. Chunk %d not found on the node within a depth. Upload node: %s Chunk: %s Node within a depth: %s\n", i, j, overlays[i].String(), chunk.Address().String(), n)
					continue
				}
				fmt.Printf("Upload node %d. Chunk %d found on the node within a depth node. Upload node: %s Chunk: %s Node within a depth: %s\n", i, j, overlays[i].String(), chunk.Address().String(), n)
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

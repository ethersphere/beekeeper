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
				chunk            bee.Chunk
				err              error
				replicatingNodes []swarm.Address
				nnRep            int
			)

			chunk, err = bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			addr, err := c.Nodes[i].UploadBytes(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			fmt.Printf("Uploaded chunk %s\n", addr.String())

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
			for _, v := range topology.Bins {
				for _, peer := range v.ConnectedPeers {
					peer := peer
					pidx := findIndex(overlays, peer)
					pivotTopology := topologies[pidx]
					pivotDepth := pivotTopology.Depth
					if pivotPo := int(swarm.Proximity(addr.Bytes(), peer.Bytes())); pivotPo >= pivotDepth {
						// chunk within replicating node depth
						if findIndex(replicatingNodes, peer) == -1 {
							replicatingNodes = append(replicatingNodes, peer)
							nnRep++
						}
					}
				}
			}

			if len(replicatingNodes) == 0 {
				fmt.Printf("Upload node %d. Chunk: %d. Chunk does not have any designated replicators.\n", i, j)
				return errPullSync
			}

			fmt.Printf("Chunk should be on %d nodes. %d within depth\n", len(replicatingNodes), nnRep)
			for _, n := range replicatingNodes {
				ni := findIndex(overlays, n)
				var (
					synced bool
					err    error
				)

				for t := 1; t < 5; t++ {
					time.Sleep(2 * time.Duration(t) * time.Second)
					synced, err = c.Nodes[ni].HasChunk(ctx, addr)
					if err != nil {
						return fmt.Errorf("node %d: %w", ni, err)
					}
					if synced {
						break
					}
					fmt.Printf("Upload node %d. Chunk %d not found on node. Upload node: %s Chunk: %s Pivot: %s. Retrying...\n", i, j, overlays[i].String(), addr.String(), n)
				}
				if !synced {
					return fmt.Errorf("Upload node %d. Chunk %d not found on node. Upload node: %s Chunk: %s Pivot: %s", i, j, overlays[i].String(), addr.String(), n)
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

// findIndex returns index of a given swarm.Address in a given set of swarm.Addresses, or -1 if not found
func findIndex(overlays []swarm.Address, addr swarm.Address) int {
	for i, a := range overlays {
		if addr.Equal(a) {
			return i
		}
	}
	return -1
}

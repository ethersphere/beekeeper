package pullsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// compile check whether Check implements interface
var _ check.Check = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() check.Check {
	return &Check{}
}

// Options represents check options
type Options struct {
	ChunksPerNode              int
	NodeGroup                  string // TODO: support multi node group cluster
	ReplicationFactorThreshold int
	Seed                       int64
	UploadNodeCount            int
}

var errPullSync = errors.New("pull sync")

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	var (
		rnds                   = random.PseudoGenerators(o.Seed, o.UploadNodeCount)
		totalReplicationFactor float64
	)

	fmt.Printf("Seed: %d\n", o.Seed)

	ng := cluster.NodeGroup(o.NodeGroup)
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
				chunk bee.Chunk
				err   error
				nnRep int
			)
			replicatingNodes := make(map[string]swarm.Address)

			chunk, err = bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			addr, err := ng.NodeClient(nodeName).UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			fmt.Printf("Uploaded chunk %s\n", addr.String())

			// check closest and NN replication (non-nn replication is not realistic)
			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			fmt.Printf("Upload node %s. Chunk: %d. Closest: %s %s\n", nodeName, j, closestName, closestAddress.String())

			topology, err := ng.NodeClient(closestName).Topology(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", closestName, err)
			}
			for _, v := range topology.Bins {
				for _, peer := range v.ConnectedPeers {
					peer := peer
					pidx := findName(overlays, peer)
					pivotTopology := topologies[pidx]
					pivotDepth := pivotTopology.Depth
					if pivotPo := int(swarm.Proximity(addr.Bytes(), peer.Bytes())); pivotPo >= pivotDepth {
						// chunk within replicating node depth
						if len(findName(replicatingNodes, peer)) == 0 {
							replicatingNodes[findName(overlays, peer)] = peer
							nnRep++
						}
					}
				}
			}

			if len(replicatingNodes) == 0 {
				fmt.Printf("Upload node %s. Chunk: %d. Chunk does not have any designated replicators.\n", nodeName, j)
				return errPullSync
			}

			fmt.Printf("Chunk should be on %d nodes. %d within depth\n", len(replicatingNodes), nnRep)
			for _, n := range replicatingNodes {
				ni := findName(overlays, n)
				var (
					synced bool
					err    error
				)

				for t := 1; t < 5; t++ {
					time.Sleep(2 * time.Duration(t) * time.Second)
					synced, err = ng.NodeClient(ni).HasChunk(ctx, chunk.Address())
					if err != nil {
						return fmt.Errorf("node %s: %w", ni, err)
					}
					if synced {
						break
					}
					fmt.Printf("Upload node %s. Chunk %d not found on node. Upload node: %s Chunk: %s Pivot: %s. Retrying...\n", nodeName, j, overlays[nodeName].String(), chunk.Address().String(), n)
				}
				if !synced {
					return fmt.Errorf("upload node %s. Chunk %d not found on node. Upload node: %s Chunk: %s Pivot: %s", nodeName, j, overlays[nodeName].String(), chunk.Address().String(), n)
				}
			}

			rf, err := cluster.GlobalReplicationFactor(ctx, chunk.Address())
			if err != nil {
				return fmt.Errorf("replication factor: %w", err)
			}

			if rf < o.ReplicationFactorThreshold {
				return fmt.Errorf("chunk %s has low replication factor. got %d want %d", chunk.Address().String(), rf, o.ReplicationFactorThreshold)
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

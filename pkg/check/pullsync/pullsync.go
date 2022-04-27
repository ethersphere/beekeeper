package pullsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/ethersphere/bee/pkg/swarm"
)

// Options represents check options
type Options struct {
	ChunksPerNode              int // number of chunks to upload per node
	GasPrice                   string
	PostageAmount              int64
	PostageLabel               string
	ReplicationFactorThreshold int // minimal replication factor per chunk
	Seed                       int64
	UploadNodeCount            int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:              1,
		GasPrice:                   "",
		PostageAmount:              1,
		PostageLabel:               "test-label",
		ReplicationFactorThreshold: 2,
		Seed:                       random.Int64(),
		UploadNodeCount:            1,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

var errPullSync = errors.New("pull sync")

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	var (
		rnds                   = random.PseudoGenerators(o.Seed, o.UploadNodeCount)
		totalReplicationFactor float64
	)

	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	topologies, err := cluster.FlattenTopologies(ctx)
	if err != nil {
		return err
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.NodeNames()
	for i := 0; i < o.UploadNodeCount; i++ {

		nodeName := sortedNodes[i]
		client := clients[nodeName]

		batchID, err := client.CreatePostageBatch(ctx, o.PostageAmount, bee.MinimumBatchDepth, o.GasPrice, o.PostageLabel, false)
		if err != nil {
			return fmt.Errorf("node %s: created batched id %w", nodeName, err)
		}
		fmt.Printf("node %s: created batched id %s\n", nodeName, batchID)

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

			addr, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
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

			topology, err := clients[closestName].Topology(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", closestName, err)
			}
			for _, v := range topology.Bins {
				for _, peer := range v.ConnectedPeers {
					peer := peer.Address
					pidx, found := findName(overlays, peer)
					if !found {
						return fmt.Errorf("1: not found in overlays: %v", peer)
					}
					pivotTopology := topologies[pidx]
					pivotDepth := pivotTopology.Depth
					if pivotPo := int(swarm.Proximity(addr.Bytes(), peer.Bytes())); pivotPo >= pivotDepth {
						// chunk within replicating node depth
						_, found := findName(replicatingNodes, peer)
						if !found {
							oName, found := findName(overlays, peer)
							if !found {
								return fmt.Errorf("2: not found in overlays: %v", peer)
							}
							replicatingNodes[oName] = peer
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
				ni, found := findName(overlays, n)
				if !found {
					return fmt.Errorf("not found: %v", n)
				}

				var (
					synced bool
					err    error
				)

				for t := 1; t < 5; t++ {
					time.Sleep(2 * time.Duration(t) * time.Second)
					synced, err = clients[ni].HasChunk(ctx, chunk.Address())
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
func findName(nodes map[string]swarm.Address, addr swarm.Address) (string, bool) {
	for n, a := range nodes {
		if addr.Equal(a) {
			return n, true
		}
	}

	return "", false
}

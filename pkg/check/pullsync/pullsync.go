package pullsync

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			if err := c.Nodes[i].UploadChunk(ctx, &chunk); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			closest, err := chunk.ClosestNode(overlays)
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			index := findIndex(overlays, closest)
			fmt.Printf("Upload node %d. Chunk: %d. Closest: %d %s\n", i, j, index, closest.String())

			topolgy, err := c.Nodes[index].Topology(ctx)
			if err != nil {
				return fmt.Errorf("node %d: %w", index, err)
			}

			po := swarm.Proximity(chunk.Address().Bytes(), closest.Bytes())

			if po < topolgy.Depth {
				fmt.Printf("Upload node %d. Chunk: %d. Chunk does not fall within a depth. Proximity: %d Depth: %d\n", i, j, po, topolgy.Depth)
				// TODO:  add indication whether a chunk does not fall within any node's depth in the cluster
				return errPullSync
			}
			fmt.Printf("Upload node %d. Chunk: %d. Chunk falls within a depth of node %d. Proximity: %d Depth: %d\n", i, j, index, po, topolgy.Depth)

			var nodesWithinDepth []swarm.Address
			for k, v := range topolgy.Bins {

				bin, err := strconv.Atoi(strings.Split(k, "_")[1])
				if err != nil {
					return fmt.Errorf("node %d: %w", i, err)
				}

				if bin >= topolgy.Depth {
					nodesWithinDepth = append(nodesWithinDepth, v.ConnectedPeers...)
				}
			}

			time.Sleep(10 * time.Second)
			for _, n := range nodesWithinDepth {
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

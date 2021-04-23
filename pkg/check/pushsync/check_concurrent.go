package pushsync

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// CheckConcurrent uploads given chunks concurrently on cluster and checks pushsync ability of the cluster
func CheckConcurrent(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	clients, err := c.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := c.NodeNames()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]

		var chunkResults []chunkStreamMsg
		for m := range chunkStream(ctx, clients[nodeName], rnds[i], o.ChunksPerNode) {
			chunkResults = append(chunkResults, m)
		}
		for j, c := range chunkResults {
			fmt.Println(i, j, c.Chunk.Size(), c.Error)
		}
	}

	return
}

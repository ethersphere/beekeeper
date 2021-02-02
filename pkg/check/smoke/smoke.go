package smoke

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents smoke test options
type Options struct {
	NodeGroup string
	Runs      int // how many runs to do
	Bytes     int // how many bytes to upload each time
	Seed      int64
}

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c *bee.Cluster, o Options) error {
	fmt.Printf("seed: %d\n", o.Seed)
	var (
		ctx         = context.Background()
		rnd         = random.PseudoGenerator(o.Seed)
		ng          = c.NodeGroup(o.NodeGroup)
		r           = rand.New(rand.NewSource(o.Seed))
		sortedNodes = ng.NodesSorted()
	)

	for i := 0; i < o.Runs; i++ {
		uploader := r.Intn(len(sortedNodes))
		nodeName := sortedNodes[i]
		fmt.Printf("run %d, uploader node is: %s\n", i, nodeName)

		data := make([]byte, o.Bytes)
		if _, err := rnd.Read(data); err != nil {
			return fmt.Errorf("create random data: %w", err)
		}

		addr, err := ng.NodeClient(nodeName).UploadBytes(ctx, data, api.UploadOptions{Pin: false})
		if err != nil {
			return fmt.Errorf("upload to node %s: %w", nodeName, err)
		}

		fmt.Printf("uploaded %d bytes successfully, hash %s\n", len(data), addr.String())

		// pick a random different node and try to download the content
		n := randNot(r, len(sortedNodes), uploader)
		downloadNode := sortedNodes[n]
		fmt.Printf("trying to download from node: %s\n", downloadNode)

		dd, err := ng.NodeClient(downloadNode).DownloadBytes(ctx, addr)
		if err != nil {
			return fmt.Errorf("download from node %s: %w", nodeName, err)
		}

		if !bytes.Equal(data, dd) {
			return fmt.Errorf("download data mismatch")
		}
	}
	fmt.Println("smoke test completed successfully")
	return nil
}

func randNot(r *rand.Rand, l, not int) int {
	for {
		pick := r.Intn(l)
		if pick != not {
			return pick
		}
	}
}

package smoke

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents smoke test options
type Options struct {
	NodeGroup       string
	UploadNodeCount int
	Runs            int // how many runs to do
	Bytes           int // how many bytes to upload each time
	Timeout         time.Duration
	Seed            int64
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
		nodeName := sortedNodes[uploader]

		fmt.Printf("run %d, uploader node is: %s\n", i, nodeName)

		tr, err := ng.NodeClient(nodeName).CreateTag(ctx)
		if err != nil {
			return fmt.Errorf("get tag from node %s: %w", nodeName, err)
		}

		data := make([]byte, o.Bytes)
		if _, err := rnd.Read(data); err != nil {
			return fmt.Errorf("create random data: %w", err)
		}

		addr, err := ng.NodeClient(nodeName).UploadBytes(ctx, data, api.UploadOptions{Pin: false, Tag: tr.Uid})
		if err != nil {
			return fmt.Errorf("upload to node %s: %w", nodeName, err)
		}

		ctx, cancel := context.WithTimeout(ctx, o.Timeout)
		defer cancel()

		err = ng.NodeClient(nodeName).WaitSync(ctx, tr.Uid)
		if err != nil {
			return fmt.Errorf("sync with node %s: %w", nodeName, err)
		}

		// pick a random different node and try to download the content
		n := randNot(r, len(sortedNodes), uploader)
		downloadNode := sortedNodes[n]

		dd, err := ng.NodeClient(downloadNode).DownloadBytes(ctx, addr)
		if err != nil {
			return fmt.Errorf("download from node %s: %w", nodeName, err)
		}

		if !bytes.Equal(data, dd) {
			return fmt.Errorf("download data mismatch")
		}

		fmt.Printf("Downloaded successfully from node: %s\n", downloadNode)
	}
	fmt.Println("smoke test completed successfully")
	return nil
}

func randNot(r *rand.Rand, l, not int) int {
	if l < 2 {
		fmt.Println("Warning: downloading from same node!")
		return 0
	}
	for {
		pick := r.Intn(l)
		if pick != not {
			return pick
		}
	}
}

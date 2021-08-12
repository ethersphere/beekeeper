package smoke

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	orchestrationK8S "github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// TODO: remove need for node group, use whole cluster instead

// Options represents smoke test options
type Options struct {
	Bytes           int // how many bytes to upload each time
	NodeGroup       string
	Runs            int // how many runs to do
	Seed            int64
	Timeout         time.Duration
	UploadNodeCount int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		Bytes:           0,
		NodeGroup:       "bee",
		Runs:            1,
		Seed:            0,
		Timeout:         1 * time.Second,
		UploadNodeCount: 1,
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

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func (c *Check) Run(ctx context.Context, cluster *orchestrationK8S.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	fmt.Printf("seed: %d\n", o.Seed)

	ng, err := cluster.NodeGroup(o.NodeGroup)
	if err != nil {
		return err
	}

	var (
		rnd         = random.PseudoGenerator(o.Seed)
		r           = rand.New(rand.NewSource(o.Seed))
		sortedNodes = ng.NodesSorted()
	)

	for i := 0; i < o.Runs; i++ {
		uploader := r.Intn(len(sortedNodes))
		nodeName := sortedNodes[uploader]
		uClient, err := ng.NodeClient(nodeName)
		if err != nil {
			return err
		}

		fmt.Printf("run %d, uploader node is: %s\n", i, nodeName)

		tr, err := uClient.CreateTag(ctx)
		if err != nil {
			return fmt.Errorf("get tag from node %s: %w", nodeName, err)
		}

		data := make([]byte, o.Bytes)
		if _, err := rnd.Read(data); err != nil {
			return fmt.Errorf("create random data: %w", err)
		}

		addr, err := uClient.UploadBytes(ctx, data, api.UploadOptions{Pin: false, Tag: tr.Uid})
		if err != nil {
			return fmt.Errorf("upload to node %s: %w", nodeName, err)
		}

		ctx, cancel := context.WithTimeout(ctx, o.Timeout)
		defer cancel()

		err = uClient.WaitSync(ctx, tr.Uid)
		if err != nil {
			return fmt.Errorf("sync with node %s: %w", nodeName, err)
		}

		// pick a random different node and try to download the content
		n := randNot(r, len(sortedNodes), uploader)
		downloadNode := sortedNodes[n]
		dClient, err := ng.NodeClient(downloadNode)
		if err != nil {
			return err
		}

		dd, err := dClient.DownloadBytes(ctx, addr)
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

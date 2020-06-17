package file

import (
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents pushsync check options
type Options struct {
	UploadNodeCount int
	ChunksPerNode   int
	Seed            int64
}

var errFile = errors.New("file")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c bee.Cluster, o Options) (err error) {
	fmt.Println("file")
	return
}

package settlements

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents settlements check options
type Options struct {
	UploadNodeCount int
	FileName        string
	FileSize        int64
	Seed            int64
}

// Check executes settlements check
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	fmt.Println("settlements")
	return
}

package pushsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	FileSize        int64
	FilesPerNode    int
	MetricsPusher   *push.Pusher
	Mode            string
	NodeGroup       string        // TODO: support multi node group cluster
	Retries         int           // number of reties on problems
	RetryDelay      time.Duration // retry delay duration
	Seed            int64
	UploadNodeCount int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:   1,
		FileSize:        1 * 1024 * 1024, // 1mb
		FilesPerNode:    1,
		MetricsPusher:   nil,
		Mode:            "default",
		NodeGroup:       "bee",
		Retries:         5,
		RetryDelay:      1 * time.Second,
		Seed:            random.Int64(),
		UploadNodeCount: 1,
	}
}

// compile check whether Check implements interface
var _ check.Check = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() check.Check {
	return &Check{}
}

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	switch o.Mode {
	case "chunks":
		return checkChunks(ctx, cluster, o)
	case "files":
		return checkFiles(ctx, cluster, o)
	default:
		return defaultCheck(ctx, cluster, o)
	}
}

var errPushSync = errors.New("push sync")

// defaultCheck uploads given chunks on cluster and checks pushsync ability of the cluster
func defaultCheck(ctx context.Context, c *bee.Cluster, o Options) error {
	fmt.Println("running pushsync")
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("seed: %d\n", o.Seed)

	if o.MetricsPusher != nil {
		o.MetricsPusher.Collector(uploadedCounter)
		o.MetricsPusher.Collector(uploadTimeGauge)
		o.MetricsPusher.Collector(uploadTimeHistogram)
		o.MetricsPusher.Collector(syncedCounter)
		o.MetricsPusher.Collector(notSyncedCounter)
		o.MetricsPusher.Format(expfmt.FmtText)
	}

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			t0 := time.Now()
			addr, err := ng.NodeClient(nodeName).UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)
			fmt.Printf("uploaded chunk %s to node %s\n", addr.String(), nodeName)

			uploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[nodeName].String(), addr.String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			fmt.Printf("closest node %s overlay %s\n", closestName, closestAddress)

			checkRetryCount := 0

			for {
				checkRetryCount++
				if checkRetryCount > o.Retries {
					return fmt.Errorf("exceeded number of retries")
				}

				time.Sleep(o.RetryDelay)
				synced, err := ng.NodeClient(closestName).HasChunk(ctx, addr)
				if err != nil {
					return fmt.Errorf("node %s: %w", nodeName, err)
				}
				if !synced {
					notSyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					fmt.Printf("node %s overlay %s chunk %s not found on the closest node. retrying...\n", closestName, overlays[closestName], addr.String())
					continue
				}

				syncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("node %s overlay %s chunk %s found on the closest node.\n", closestName, overlays[closestName], addr.String())

				// check succeeded
				break
			}

			if o.MetricsPusher != nil {
				if err := o.MetricsPusher.Push(); err != nil {
					return fmt.Errorf("node %s: %v", nodeName, err)
				}
			}
		}
	}

	return nil
}

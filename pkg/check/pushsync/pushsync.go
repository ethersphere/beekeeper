package pushsync

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	ChunksPerNode     int
	GasPrice          string
	Mode              string
	PostageAmount     int64
	PostageDepth      uint64
	PostageLabel      string
	Retries           int           // number of reties on problems
	RetryDelay        time.Duration // retry delay duration
	Seed              int64
	UploadNodeCount   int
	ExcludeNodeGroups []string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:     1,
		GasPrice:          "",
		Mode:              "default",
		PostageAmount:     1000,
		PostageDepth:      16,
		PostageLabel:      "test-label",
		Retries:           5,
		RetryDelay:        1 * time.Second,
		Seed:              random.Int64(),
		UploadNodeCount:   1,
		ExcludeNodeGroups: []string{},
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		metrics: newMetrics(),
		logger:  logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	switch o.Mode {
	case "chunks":
		return checkChunks(ctx, cluster, o, c.logger)
	case "light-chunks":
		return checkLightChunks(ctx, cluster, o, c.logger)
	default:
		return c.defaultCheck(ctx, cluster, o)
	}
}

// defaultCheck uploads given chunks on cluster and checks pushsync ability of the cluster
func (c *Check) defaultCheck(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	c.logger.Infof("seed: %d", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx)
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

		batchID, err := client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", nodeName, err)
		}
		c.logger.Infof("node %s: batch id %s", nodeName, batchID)

		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i], c.logger)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			t0 := time.Now()
			addr, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false, BatchID: batchID})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)
			c.logger.Infof("uploaded chunk %s to node %s", addr.String(), nodeName)

			c.metrics.UploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			c.metrics.UploadTimeGauge.WithLabelValues(overlays[nodeName].String(), addr.String()).Set(d0.Seconds())
			c.metrics.UploadTimeHistogram.Observe(d0.Seconds())

			closestName, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			c.logger.Infof("closest node %s overlay %s", closestName, closestAddress)

			checkRetryCount := 0

			for {
				checkRetryCount++
				if checkRetryCount > o.Retries {
					return fmt.Errorf("exceeded number of retries")
				}

				time.Sleep(o.RetryDelay)
				node := clients[closestName]
				synced, err := node.HasChunk(ctx, addr)
				if err != nil {
					return fmt.Errorf("node %s: %w", nodeName, err)
				}
				if !synced {
					c.metrics.NotSyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					c.logger.Infof("node %s overlay %s chunk %s not found on the closest node. retrying...", closestName, overlays[closestName], addr.String())
					continue
				}

				c.metrics.SyncedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				c.logger.Infof("node %s overlay %s chunk %s found on the closest node.", closestName, overlays[closestName], addr.String())

				// check succeeded
				break
			}
		}
	}

	return nil
}

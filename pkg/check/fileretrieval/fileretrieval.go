package fileretrieval

import (
	"bytes"
	"context"
	"errors"
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
	FileName        string
	FileSize        int64
	FilesPerNode    int
	Full            bool
	GasPrice        string
	PostageAmount   int64
	PostageTTL      time.Duration
	PostageLabel    string
	Seed            int64
	UploadNodeCount int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		FileName:        "file-retrieval",
		FileSize:        1 * 1024 * 1024, // 1mb
		FilesPerNode:    1,
		Full:            false,
		GasPrice:        "",
		PostageAmount:   1,
		PostageTTL:      24 * time.Hour,
		PostageLabel:    "test-label",
		Seed:            0,
		UploadNodeCount: 1,
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

	if o.Full {
		return c.fullCheck(ctx, cluster, o)
	}

	return c.defaultCheck(ctx, cluster, o)
}

var errFileRetrieval = errors.New("file retrieval")

// defaultCheck uploads files on cluster and downloads them from the last node in the cluster
func (c *Check) defaultCheck(ctx context.Context, cluster orchestration.Cluster, o Options) (err error) {
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	c.logger.Infof("Seed: %d", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.NodeNames()
	lastNodeName := sortedNodes[len(sortedNodes)-1]
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.FilesPerNode; j++ {
			file := bee.NewRandomFile(rnds[i], fmt.Sprintf("%s-%d-%d", o.FileName, i, j), o.FileSize)

			depth := 2 + bee.EstimatePostageBatchDepth(file.Size())
			batchID, err := clients[nodeName].CreatePostageBatch(ctx, o.PostageAmount, depth, o.PostageLabel, false)
			if err != nil {
				return fmt.Errorf("node %s: created batched id %w", nodeName, err)
			}
			c.logger.Infof("node %s: created batched id %s", nodeName, batchID)

			t0 := time.Now()

			client := clients[nodeName]
			if err := client.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID}); err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			d0 := time.Since(t0)

			c.metrics.UploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			c.metrics.UploadTimeGauge.WithLabelValues(overlays[nodeName].String(), file.Address().String()).Set(d0.Seconds())
			c.metrics.UploadTimeHistogram.Observe(d0.Seconds())

			time.Sleep(1 * time.Second)
			t1 := time.Now()

			client = clients[lastNodeName]

			size, hash, err := client.DownloadFile(ctx, file.Address(), nil)
			if err != nil {
				return fmt.Errorf("node %s: %w", lastNodeName, err)
			}
			d1 := time.Since(t1)

			c.metrics.DownloadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			c.metrics.DownloadTimeGauge.WithLabelValues(overlays[nodeName].String(), file.Address().String()).Set(d1.Seconds())
			c.metrics.DownloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(file.Hash(), hash) {
				c.metrics.NotRetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				c.logger.Infof("Node %s. File %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s File: %s", nodeName, j, file.Size(), size, overlays[nodeName].String(), file.Address().String())
				return errFileRetrieval
			}

			c.metrics.RetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			c.logger.Infof("Node %s. File %d retrieved successfully. Node: %s File: %s", nodeName, j, overlays[nodeName].String(), file.Address().String())
		}
	}

	return
}

// fullCheck uploads files on cluster and downloads them from the all nodes in the cluster
func (c *Check) fullCheck(ctx context.Context, cluster orchestration.Cluster, o Options) (err error) {
	c.logger.Info("running file retrieval (full mode)")

	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	c.logger.Infof("Seed: %d", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.NodeNames()

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.FilesPerNode; j++ {
			file := bee.NewRandomFile(rnds[i], fmt.Sprintf("%s-%d-%d", o.FileName, i, j), o.FileSize)

			depth := 2 + bee.EstimatePostageBatchDepth(file.Size())
			batchID, err := clients[nodeName].CreatePostageBatch(ctx, o.PostageAmount, depth, o.PostageLabel, false)
			if err != nil {
				return fmt.Errorf("node %s: created batched id %w", nodeName, err)
			}
			c.logger.Infof("node %s: created batched id %s", nodeName, batchID)

			t0 := time.Now()
			if err := clients[nodeName].UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID}); err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)

			c.metrics.UploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			c.metrics.UploadTimeGauge.WithLabelValues(overlays[nodeName].String(), file.Address().String()).Set(d0.Seconds())
			c.metrics.UploadTimeHistogram.Observe(d0.Seconds())

			time.Sleep(1 * time.Second)
			for n, nc := range clients {
				if n == nodeName {
					continue
				}

				t1 := time.Now()
				size, hash, err := nc.DownloadFile(ctx, file.Address(), nil)
				if err != nil {
					return fmt.Errorf("node %s: %w", n, err)
				}
				d1 := time.Since(t1)

				c.metrics.DownloadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				c.metrics.DownloadTimeGauge.WithLabelValues(overlays[nodeName].String(), file.Address().String()).Set(d1.Seconds())
				c.metrics.DownloadTimeHistogram.Observe(d1.Seconds())

				if !bytes.Equal(file.Hash(), hash) {
					c.metrics.NotRetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					c.logger.Infof("Node %s. File %d not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d Node: %s Download node: %s File: %s", nodeName, j, n, file.Size(), size, overlays[nodeName].String(), overlays[n].String(), file.Address().String())
					return errFileRetrieval
				}

				c.metrics.RetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				c.logger.Infof("Node %s. File %d retrieved successfully from node %s. Node: %s Download node: %s File: %s", nodeName, j, n, overlays[nodeName].String(), overlays[n].String(), file.Address().String())
			}
		}
	}

	return
}

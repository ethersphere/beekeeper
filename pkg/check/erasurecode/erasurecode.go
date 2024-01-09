package erasurecode

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"time"
)

// Options represents smoke test options
type Options struct {
	ContentSize            int64
	RndSeed                int64
	PostageAmount          int64
	PostageDepth           uint64
	TxOnErrWait            time.Duration
	RxOnErrWait            time.Duration
	NodesSyncWait          time.Duration
	ChunkRetrievalTimeouts []string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ContentSize:            5000000,
		RndSeed:                time.Now().UnixNano(),
		PostageAmount:          50_000_000,
		PostageDepth:           24,
		TxOnErrWait:            10 * time.Second,
		RxOnErrWait:            10 * time.Second,
		NodesSyncWait:          time.Minute,
		ChunkRetrievalTimeouts: []string{"100ms", "200ms", "300ms", "500ms", "1s", "1.5s"},
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
		metrics: newMetrics("check_erasure_code", []string{"level", "strategy", "timeout", "fallback"}),
		logger:  logger,
	}
}

// Run creates file of specified size that is uploaded and downloaded.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	var timeouts []time.Duration
	for _, t := range o.ChunkRetrievalTimeouts {
		timeout, err := time.ParseDuration(t)
		if err != nil {
			return fmt.Errorf("invalid timeout %s: %w", t, err)
		}
		timeouts = append(timeouts, timeout)
	}

	c.logger.Info("random seed: ", o.RndSeed)
	c.logger.Info("content size: ", o.ContentSize)
	time.Sleep(5 * time.Second) // Wait for the nodes to warmup.

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	for rLevel := 0; rLevel < 5; rLevel++ {
		rnd := random.PseudoGenerator(o.RndSeed)
		perm := rnd.Perm(cluster.Size())
		var txIdx, rxIdx int
		for {
			txIdx = perm[0]
			rxIdx = perm[1]
			if txIdx != rxIdx {
				break
			}
		}

		nn := cluster.NodeNames()
		txName := nn[txIdx]
		rxName := nn[rxIdx]

		addr, txHash, err := upload(ctx, c.logger, clients, txName, o, rLevel)
		if err != nil {
			c.logger.Errorf("upload failed: node=%s, level=%d, err=%v. skipping", txName, rLevel, err)
			continue
		}

		c.logger.Infof("file uploaded. node=%s, level=%d, address=%s", txName, rLevel, addr)
		for _, timeout := range timeouts {
			cache, fallback := false, false
			for strategy := 0; strategy < 4; strategy++ {
				labelValues := []string{fmt.Sprintf("%d", rLevel), fmt.Sprintf("%d", strategy), timeout.String(), fmt.Sprintf("%t", fallback)}
				c.metrics.DownloadAttempts.WithLabelValues(labelValues...).Inc()
				c.logger.Infof("downloading file. address=%s, level=%d, strategy=%d, cache=%t, fallback=%t, timeout=%s", addr.String(), rLevel, strategy, cache, fallback, timeout.String())
				_, rxHash, err := clients[rxName].DownloadFile(ctx, addr, &api.DownloadOptions{
					Cache:                  &cache,
					RedundancyStrategy:     &strategy,
					RedundancyFallbackMode: &fallback,
					ChunkRetrievalTimeout:  &timeout,
				})
				if strategy == 0 {
					strategy-- // repeat for strategy 0 with fallback=true
					fallback = true
				}

				if err != nil {
					c.logger.Errorf("download failed. address=%s, node=%s, level=%d, strategy=%d, cache=%t, fallback=%t, timeout=%s, err=%v", addr.String(), rxName, rLevel, strategy, cache, fallback, timeout.String(), err)
					c.metrics.DownloadErrors.WithLabelValues(labelValues...).Inc()
					continue
				}
				if !bytes.Equal(txHash, rxHash) {
					c.logger.Errorf("hashes dont match. address=%s, node=%s, level=%d, strategy=%d, cache=%t, fallback=%t, timeout=%s", addr.String(), rxName, rLevel, strategy, cache, fallback, timeout.String())
					c.metrics.DownloadErrors.WithLabelValues(labelValues...).Inc()
					continue
				}
				c.logger.Infof("download successful. address=%s, level=%d, strategy=%d, cache=%t, fallback=%t, timeout=%s", addr.String(), rLevel, strategy, cache, fallback, timeout.String())
			}
		}
	}
	return nil
}

func upload(ctx context.Context, logger logging.Logger, clients map[string]*bee.Client, cName string, o Options, rLevel int) (swarm.Address, []byte, error) {
	for retries := 0; retries < 3; retries++ {
		select {
		case <-time.After(o.TxOnErrWait):
		}
		batchID, err := clients[cName].GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, "erasure-code")
		if err != nil {
			logger.Warningf("level: %d. create new batch: %v", rLevel, err)
			continue
		}

		logger.Infof("node %s: uploading data, batch id %s", cName, batchID)
		rnd := random.PseudoGenerator(time.Now().UnixNano())
		file := bee.NewRandomFile(rnd, fmt.Sprintf("%d", time.Now().UnixNano()), o.ContentSize)
		err = clients[cName].UploadFile(ctx, &file, api.UploadOptions{Pin: false, BatchID: batchID, Direct: true, RedundancyLevel: &rLevel})
		if err != nil {
			logger.Warningf("level: %d. upload file to the node %s: %w", rLevel, cName, err)
			continue
		}
		return file.Address(), file.Hash(), nil
	}
	return swarm.ZeroAddress, nil, fmt.Errorf("level: %d. unable to upload", rLevel)
}

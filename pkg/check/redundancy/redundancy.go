package redundancy

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

type File struct {
	Ref   string `yaml:"ref"`
	Level int    `yaml:"level"`
}

type Options struct {
	Files   []File
	RndSeed int64
}

func NewDefaultOptions() Options {
	return Options{
		RndSeed: time.Now().UnixNano(),
	}
}

var _ beekeeper.Action = (*Check)(nil)

type Check struct {
	metrics metrics
	logger  logging.Logger
}

func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger:  logger,
		metrics: newMetrics("check_redundancy", []string{"ref", "level", "strategy", "fallback"}),
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o interface{}) error {
	opts, ok := o.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(opts.RndSeed)
	for _, fileOpt := range opts.Files {
		ref, err := swarm.ParseHexAddress(fileOpt.Ref)
		if err != nil {
			return fmt.Errorf("parse hex address (%s): %w", fileOpt.Ref, err)
		}

		for strategy := 0; strategy < 4; strategy++ {
			for fallback := 0; fallback < 2; fallback++ {
				start, cache := time.Now(), false
				node := cluster.Nodes()[cluster.NodeNames()[rnd.Intn(len(cluster.NodeNames()))]]
				downloadOpts := &api.DownloadOptions{Cache: &cache, RedundancyStrategy: &strategy, RedundancyFallbackMode: &fallback}

				lvs := []string{ref.String(), fmt.Sprintf("%d", fileOpt.Level), fmt.Sprintf("%d", strategy), fmt.Sprintf("%d", fallback)}
				c.metrics.DownloadAttempts.WithLabelValues(lvs...).Inc()
				c.logger.Infof("downloading file: node=%s, ref=%s, level=%d, strategy=%d, fallback=%d", node.Name(), ref.String(), fileOpt.Level, strategy, fallback)
				_, _, err = node.Client().DownloadFile(ctx, ref, downloadOpts)
				if err != nil {
					c.metrics.DownloadErrors.WithLabelValues(lvs...).Inc()
					c.logger.Errorf("download failed: %v", err)
					continue
				}
				dur := time.Since(start)
				c.metrics.DownloadDuration.WithLabelValues(lvs...).Observe(dur.Seconds())
				c.logger.Infof("download successful: dur=%v", dur)
			}
		}
	}
	return nil
}

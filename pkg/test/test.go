package test

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

func NewTest(logger logging.Logger) *test {
	return &test{
		logger: logger,
	}
}

type test struct {
	logger logging.Logger
}

func (t *test) Upload(ctx context.Context, bee *bee.Client, data []byte, batchID string, rLevel redundancy.Level) (swarm.Address, time.Duration, error) {
	t.logger.Infof("node %s: uploading %d bytes, batch id %s", bee.Name(), len(data), batchID)
	start := time.Now()
	addr, err := bee.UploadBytes(ctx, data, api.UploadOptions{Pin: false, BatchID: batchID, Direct: true, RLevel: rLevel})
	if err != nil {
		return swarm.ZeroAddress, 0, fmt.Errorf("upload to node %s: %w", bee.Name(), err)
	}

	txDuration := time.Since(start)
	t.logger.Infof("node %s: upload completed for %d bytes in %s", bee.Name(), len(data), txDuration)

	return addr, txDuration, nil
}

func (t *test) Download(ctx context.Context, bee *bee.Client, addr swarm.Address, rLevel redundancy.Level) ([]byte, time.Duration, error) {
	t.logger.Infof("node %s: downloading address %s", bee.Name(), addr)

	start := time.Now()

	var downloadOpts *api.DownloadOptions
	if rLevel != redundancy.NONE {
		fallbackMode := true
		downloadOpts = &api.DownloadOptions{
			RLevel:                 rLevel,
			RedundancyFallbackMode: &fallbackMode,
		}
	}

	data, err := bee.DownloadBytes(ctx, addr, downloadOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("download from node %s: %w", bee.Name(), err)
	}

	rxDuration := time.Since(start)
	t.logger.Infof("node %s: download completed for %d bytes in %s", bee.Name(), len(data), rxDuration)

	return data, rxDuration, nil
}

package test

import (
	"context"
	"fmt"
	"time"

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

func (t *test) Upload(ctx context.Context, bee *bee.Client, data []byte, batchID string) (swarm.Address, time.Duration, error) {
	t.logger.Infof("node %s: uploading data, batch id %s", bee.Name(), batchID)
	start := time.Now()
	addr, err := bee.UploadBytes(ctx, data, api.UploadOptions{Pin: false, BatchID: batchID, Direct: true})
	if err != nil {
		return swarm.ZeroAddress, 0, fmt.Errorf("upload to the node %s: %w", bee.Name(), err)
	}

	txDuration := time.Since(start)
	t.logger.Infof("node %s: upload done in %s", bee.Name(), txDuration)

	return addr, txDuration, nil
}

func (t *test) Download(ctx context.Context, bee *bee.Client, addr swarm.Address) ([]byte, time.Duration, error) {
	t.logger.Infof("node %s: downloading address %s", bee, addr)

	start := time.Now()
	data, err := bee.DownloadBytes(ctx, addr, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("download from node %s: %w", bee.Name(), err)
	}

	rxDuration := time.Since(start)
	t.logger.Infof("node %s: download done in %s", bee.Name(), rxDuration)

	return data, rxDuration, nil
}

package stampexpiry

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
	FileSize     int64
	PostageTTL   time.Duration
	PostageDepth uint64
	PostageLabel string
	PollInterval time.Duration
	MaxWait      time.Duration
	Seed         int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		FileSize:     1 * 1024 * 1024, // 1mb
		PostageTTL:   5 * time.Minute,
		PostageDepth: 17,
		PostageLabel: "stamp-expiry",
		PollInterval: 15 * time.Second,
		MaxWait:      15 * time.Minute,
		Seed:         0,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	logger logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

// Run buys a short-lived postage batch, uploads a file, waits for the batch to
// expire and verifies the batch is no longer usable.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) error {
	o, ok := opts.(Options)
	if !ok {
		return errors.New("invalid options type")
	}

	// Use a random full node instead of always the same one.
	clients, err := cluster.ShuffledFullNodeClients(ctx, random.PseudoGenerator(time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("get shuffled full node clients: %w", err)
	}
	if len(clients) == 0 {
		return errors.New("no full nodes in cluster")
	}
	node := clients[0]

	// Create a short-lived postage batch. GetOrCreateMutableBatch waits for the
	// batch to become usable (covering the ~10-block minimum) before returning.
	c.logger.Infof("node %s: creating postage batch ttl=%s depth=%d", node.Name(), o.PostageTTL, o.PostageDepth)
	batchID, err := node.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: create postage batch: %w", node.Name(), err)
	}
	c.logger.Infof("node %s: using batch %s", node.Name(), batchID)

	// Upload a file and verify it is retrievable before expiry.
	rnd := random.PseudoGenerators(o.Seed, 1)[0]
	file := bee.NewRandomFile(rnd, "stamp-expiry", o.FileSize)
	if err := node.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %s: upload file: %w", node.Name(), err)
	}
	_, hash, err := node.DownloadFile(ctx, file.Address(), nil)
	if err != nil {
		return fmt.Errorf("node %s: pre-expiry download: %w", node.Name(), err)
	}
	if !bytes.Equal(file.Hash(), hash) {
		return fmt.Errorf("node %s: pre-expiry hash mismatch", node.Name())
	}
	c.logger.Infof("node %s: uploaded and retrieved file %s", node.Name(), file.Address())

	// Wait for the batch to expire (no longer usable, or evicted entirely).
	if err := c.waitForExpiry(ctx, node, batchID, o.PollInterval, o.MaxWait); err != nil {
		return err
	}

	c.logger.Infof("stamp-expiry check passed")
	return nil
}

func (c *Check) waitForExpiry(ctx context.Context, client *bee.Client, batchID string, poll, maxWait time.Duration) error {
	c.logger.Infof("waiting for batch %s to expire (poll=%s, max=%s)", batchID, poll, maxWait)

	timeout := time.After(maxWait)
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("batch %s did not expire within %s", batchID, maxWait)
		case <-ticker.C:
			batch, err := client.PostageStamp(ctx, batchID)
			if err != nil {
				c.logger.Infof("batch %s no longer queryable (likely evicted): %v", batchID, err)
				return nil
			}
			c.logger.Infof("batch %s: TTL=%d usable=%v", batchID, batch.BatchTTL, batch.Usable)
			if batch.BatchTTL <= 0 || !batch.Usable {
				c.logger.Infof("batch %s expired", batchID)
				return nil
			}
		}
	}
}

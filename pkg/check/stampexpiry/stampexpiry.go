package stampexpiry

import (
	"bytes"
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
	FileSize      int64
	PostageAmount int64
	PostageDepth  uint64
	PostageLabel  string
	PollInterval  time.Duration
	MaxWait       time.Duration
	Seed          int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		FileSize:      1 * 1024 * 1024, // 1mb
		PostageAmount: 1000,
		PostageDepth:  17,
		PostageLabel:  "stamp-expiry-test",
		PollInterval:  5 * time.Second,
		MaxWait:       10 * time.Minute,
		Seed:          0,
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

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.FullNodeNames()
	if len(sortedNodes) == 0 {
		return fmt.Errorf("no nodes in cluster")
	}

	uploadNode := sortedNodes[0]
	client := clients[uploadNode]

	// Step 1: Create postage batch with explicit amount for controlled expiry
	c.logger.Infof("node %s: creating postage batch amount=%d depth=%d", uploadNode, o.PostageAmount, o.PostageDepth)
	batchID, err := client.CreatePostageBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel, false)
	if err != nil {
		return fmt.Errorf("node %s: create postage batch: %w", uploadNode, err)
	}
	c.logger.Infof("node %s: created batch %s", uploadNode, batchID)

	// Verify batch exists and is usable
	batch, err := client.PostageStamp(ctx, batchID)
	if err != nil {
		return fmt.Errorf("node %s: get postage stamp: %w", uploadNode, err)
	}
	if !batch.Usable {
		return fmt.Errorf("node %s: batch %s not usable after creation", uploadNode, batchID)
	}
	c.logger.Infof("node %s: batch %s usable, TTL=%d", uploadNode, batchID, batch.BatchTTL)

	// Step 2: Upload a file
	rnds := random.PseudoGenerators(o.Seed, 1)
	file := bee.NewRandomFile(rnds[0], "stamp-expiry", o.FileSize)

	c.logger.Infof("node %s: uploading file (%d bytes)", uploadNode, o.FileSize)
	if err := client.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %s: upload file: %w", uploadNode, err)
	}
	c.logger.Infof("node %s: file uploaded, address=%s", uploadNode, file.Address())

	// Step 3: Verify file is retrievable before expiry
	size, hash, err := client.DownloadFile(ctx, file.Address(), nil)
	if err != nil {
		return fmt.Errorf("node %s: pre-expiry download: %w", uploadNode, err)
	}
	if !bytes.Equal(file.Hash(), hash) {
		return fmt.Errorf("node %s: pre-expiry hash mismatch (uploaded %d, downloaded %d)", uploadNode, file.Size(), size)
	}
	c.logger.Infof("node %s: pre-expiry retrieval verified", uploadNode)

	// Step 4: Wait for the stamp to expire
	if err := c.waitForExpiry(ctx, client, batchID, o); err != nil {
		return err
	}

	// Step 5: Post-expiry checks
	return c.verifyPostExpiry(ctx, clients, sortedNodes, file, batchID)
}

func (c *Check) waitForExpiry(ctx context.Context, client *bee.Client, batchID string, o Options) error {
	c.logger.Infof("waiting for batch %s to expire (poll=%s, max=%s)", batchID, o.PollInterval, o.MaxWait)

	deadline := time.After(o.MaxWait)
	ticker := time.NewTicker(o.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("batch %s did not expire within %s", batchID, o.MaxWait)
		case <-ticker.C:
			batch, err := client.PostageStamp(ctx, batchID)
			if err != nil {
				// Batch may have been evicted entirely
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

func (c *Check) verifyPostExpiry(ctx context.Context, clients map[string]*bee.Client, nodeNames []string, file bee.File, batchID string) error {
	c.logger.Infof("verifying post-expiry state")

	// Check 1: Batch should be unusable on all nodes
	for _, name := range nodeNames {
		batch, err := clients[name].PostageStamp(ctx, batchID)
		if err != nil {
			c.logger.Infof("node %s: batch gone (expected after eviction)", name)
			continue
		}
		if batch.Usable {
			return fmt.Errorf("node %s: batch %s still usable after expiry", name, batchID)
		}
		c.logger.Infof("node %s: batch %s not usable (correct)", name, batchID)
	}

	// Check 2: Log file retrievability (soft check — GC timing is non-deterministic)
	for _, name := range nodeNames {
		_, _, err := clients[name].DownloadFile(ctx, file.Address(), nil)
		if err != nil {
			c.logger.Infof("node %s: file no longer retrievable (GC ran): %v", name, err)
		} else {
			c.logger.Infof("node %s: file still retrievable (GC hasn't run yet)", name)
		}
	}

	// Check 3: New upload with expired batch should be rejected
	uploadNode := nodeNames[0]
	rnds := random.PseudoGenerators(999, 1)
	newFile := bee.NewRandomFile(rnds[0], "should-fail", 1024)
	err := clients[uploadNode].UploadFile(ctx, &newFile, api.UploadOptions{BatchID: batchID})
	if err == nil {
		return fmt.Errorf("node %s: upload with expired batch %s should have been rejected", uploadNode, batchID)
	}
	c.logger.Infof("node %s: upload with expired batch correctly rejected: %v", uploadNode, err)

	c.logger.Infof("stamp-expiry check passed")
	return nil
}

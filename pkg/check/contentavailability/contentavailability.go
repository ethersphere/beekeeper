package contentavailability

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/file/pipeline/builder"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

// Options groups a set of options that can be set for this check.
type Options struct {
	ContentSize   int64
	GasPrice      string
	PostageAmount int64
	PostageDepth  uint64
	PostageLabel  string
	Seed          int64
}

// NewDefaultOptions returns new default options.
func NewDefaultOptions() Options {
	return Options{
		ContentSize:   1024 << 4,
		GasPrice:      "",
		PostageAmount: 1000,
		PostageDepth:  16,
		PostageLabel:  "test-label",
		Seed:          0,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(o.Seed)
	idx := rnd.Intn(cluster.Size())
	node := cluster.NodeNames()[idx]

	content := make([]byte, o.ContentSize)
	if _, err := rnd.Read(content); err != nil {
		return fmt.Errorf("unable to create content: %w", err)
	}

	store := &putterMock{exists: make(map[string]swarm.Address)}
	pipe := builder.NewPipelineBuilder(ctx, store, storage.ModePutUpload, false)
	addr, err := builder.FeedPipeline(ctx, pipe, bytes.NewBuffer(content))
	if err != nil {
		return fmt.Errorf("unable to feed pipeline: %w", err)
	}

	addresses := store.Addresses()
	if len(addresses) == 0 {
		return errors.New("empty list of addresses")
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}
	client := clients[node]

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	c.logger.Infof("%s: %s", node, overlays[node])

	batchID, err := client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: unable to create batch id: %w", node, err)
	}
	c.logger.Infof("node %s: batch id %s", node, batchID)

	contentAddr, err := client.UploadBytes(ctx, content, api.UploadOptions{BatchID: batchID})
	if err != nil {
		return fmt.Errorf("node %s: unable to upload content: %w", node, err)
	}
	c.logger.Infof("node %s: content uploaded successfully: %s", node, addr)

	time.Sleep(time.Minute) // Wait for nodes to sync.

	isRetrievable, err := client.IsRetrievable(ctx, contentAddr)
	if err != nil {
		return fmt.Errorf("node %s: unable to check if content is retrievable: %w", node, err)
	}
	if !isRetrievable {
		return fmt.Errorf("node %s: the uploaded content is not retrievable", node)
	}
	c.logger.Infof("node %s: uploaded content is retrievable", node)

	rmChAddr := addresses[len(addresses)-1]
	for node, nClient := range clients {
		if err := nClient.RemoveChunk(ctx, rmChAddr); err != nil {
			return fmt.Errorf("node %s: unable to remove chunk %s: %w", node, rmChAddr, err)
		}
		c.logger.Infof("node %s: chunk %s removed", node, rmChAddr)
	}
	isRetrievable, err = client.IsRetrievable(ctx, contentAddr)
	if err != nil {
		return fmt.Errorf("node %s: unable to check if content is retrievable: %w", node, err)
	}
	if isRetrievable {
		return fmt.Errorf("node %s: the uploaded content is retrievable", node)
	}
	c.logger.Infof("node %s: the uploaded content is not retrievable", node)

	return nil
}

type putterMock struct {
	chunks []swarm.Chunk
	exists map[string]swarm.Address
}

func (pm *putterMock) Put(_ context.Context, _ storage.ModePut, chs ...swarm.Chunk) ([]bool, error) {
	exists := make([]bool, len(chs))
	for i, ch := range chs {
		key := ch.Address().ByteString()
		if _, ok := pm.exists[key]; ok {
			exists[i] = true
			continue
		}
		pm.chunks = append(pm.chunks, ch)
		pm.exists[key] = ch.Address()
	}
	return exists, nil
}

func (pm *putterMock) Addresses() []swarm.Address {
	addresses := make([]swarm.Address, 0, len(pm.exists))
	for _, val := range pm.exists {
		addresses = append(addresses, val)
	}
	return addresses
}

package chunkstream

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	ChunksPerNode    int // number of chunks to stream per upload node
	PostageTTL       time.Duration
	PostageDepth     uint64
	PostageLabel     string
	RequestBatchSize int           // addresses per download request frame
	RequestTimeout   time.Duration // per websocket read or write
	Seed             int64
	SkipNotFound     bool // skip the not-found assertion
	UploadNodeCount  int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:    10,
		PostageTTL:       24 * time.Hour,
		PostageDepth:     16,
		PostageLabel:     "test-label",
		RequestBatchSize: 5,
		RequestTimeout:   60 * time.Second,
		Seed:             random.Int64(),
		SkipNotFound:     false,
		UploadNodeCount:  1,
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

var errChunkStream = errors.New("chunk stream")

// Run streams chunks onto the cluster over the /chunks/stream websocket
// endpoint and streams them back from a different node over the same endpoint,
// verifying that every requested address gets exactly one correct delivery.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	if o.ChunksPerNode < 1 {
		return fmt.Errorf("chunks-per-node must be at least 1, got %d", o.ChunksPerNode)
	}
	if o.RequestBatchSize < 1 {
		return fmt.Errorf("request-batch-size must be at least 1, got %d", o.RequestBatchSize)
	}

	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	nodes := cluster.FullNodeNames()
	if len(nodes) < 2 {
		return fmt.Errorf("at least 2 full nodes are needed, got %d", len(nodes))
	}

	for i := range o.UploadNodeCount {
		uploadNode := clients[nodes[i]]
		downloadNode := clients[nodes[(i+1)%len(nodes)]] // download from the next node

		batchID, err := uploadNode.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: create batch: %w", uploadNode.Name(), err)
		}
		c.logger.Infof("node %s: using batch %s", uploadNode.Name(), batchID)

		chunks, err := c.streamUpload(ctx, uploadNode, batchID, rnds[i], o)
		if err != nil {
			return fmt.Errorf("node %s: upload stream: %w", uploadNode.Name(), err)
		}

		if err := c.streamDownload(ctx, downloadNode, chunks, o); err != nil {
			return fmt.Errorf("node %s: download stream: %w", downloadNode.Name(), err)
		}

		if !o.SkipNotFound {
			if err := c.checkNotFound(ctx, downloadNode, rnds[i], o); err != nil {
				return fmt.Errorf("node %s: not-found delivery: %w", downloadNode.Name(), err)
			}
		}
	}

	return nil
}

// streamUpload uploads chunks over a single websocket connection and returns
// them keyed by address.
func (c *Check) streamUpload(ctx context.Context, node *bee.Client, batchID string, rnd *rand.Rand, o Options) (map[string]bee.Chunk, error) {
	stream, err := node.API().ChunkStream.NewUploadStream(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			c.logger.Debugf("node %s: closing upload stream: %v", node.Name(), err)
		}
	}()

	chunks := make(map[string]bee.Chunk, o.ChunksPerNode)

	start := time.Now()
	for range o.ChunksPerNode {
		chunk, err := bee.NewRandomChunk(rnd, c.logger)
		if err != nil {
			return nil, fmt.Errorf("create chunk: %w", err)
		}

		if err := stream.Upload(chunk.Data(), o.RequestTimeout); err != nil {
			c.metrics.UploadErrorCounter.WithLabelValues(node.Name()).Inc()
			return nil, fmt.Errorf("chunk %s: %w", chunk.Address(), err)
		}

		chunks[chunk.Address().String()] = chunk
		c.metrics.UploadedCounter.WithLabelValues(node.Name()).Inc()
	}
	elapsed := time.Since(start)

	c.metrics.UploadTimeHistogram.Observe(elapsed.Seconds())
	c.logger.Infof("node %s: streamed %d chunks in %v", node.Name(), len(chunks), elapsed)

	return chunks, nil
}

// streamDownload requests every chunk over a single websocket connection and
// verifies the deliveries. Responses arrive out of order, so they are matched
// by the address each delivery carries.
func (c *Check) streamDownload(ctx context.Context, node *bee.Client, chunks map[string]bee.Chunk, o Options) error {
	stream, err := node.API().ChunkStream.NewDownloadStream(ctx)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			c.logger.Debugf("node %s: closing download stream: %v", node.Name(), err)
		}
	}()

	addrs := make([]swarm.Address, 0, len(chunks))
	for _, chunk := range chunks {
		addrs = append(addrs, chunk.Address())
	}

	start := time.Now()
	for i := 0; i < len(addrs); i += o.RequestBatchSize {
		end := min(i+o.RequestBatchSize, len(addrs))
		if err := stream.Request(addrs[i:end], o.RequestTimeout); err != nil {
			return err
		}
	}

	// Exactly one delivery is expected per requested address.
	pending := make(map[string]struct{}, len(chunks))
	for addr := range chunks {
		pending[addr] = struct{}{}
	}

	for len(pending) > 0 {
		delivery, err := stream.Receive(o.RequestTimeout)
		if err != nil {
			return fmt.Errorf("%d of %d deliveries outstanding: %w", len(pending), len(chunks), err)
		}

		addr := delivery.Address.String()
		chunk, known := chunks[addr]
		if !known {
			return fmt.Errorf("%w: delivery for address %s which was never requested", errChunkStream, addr)
		}
		if _, outstanding := pending[addr]; !outstanding {
			return fmt.Errorf("%w: duplicate delivery for address %s", errChunkStream, addr)
		}
		delete(pending, addr)

		if delivery.Status != api.ChunkDeliverySuccess {
			c.metrics.NotRetrievedCounter.WithLabelValues(node.Name()).Inc()
			return fmt.Errorf("%w: address %s: expected status 0x%02x, got 0x%02x", errChunkStream, addr, api.ChunkDeliverySuccess, delivery.Status)
		}
		if !bytes.Equal(delivery.Data, chunk.Data()) {
			c.metrics.NotRetrievedCounter.WithLabelValues(node.Name()).Inc()
			return fmt.Errorf("%w: address %s: downloaded %d bytes, expected %d", errChunkStream, addr, len(delivery.Data), len(chunk.Data()))
		}

		c.metrics.DownloadedCounter.WithLabelValues(node.Name()).Inc()
	}
	elapsed := time.Since(start)

	c.metrics.DownloadTimeHistogram.Observe(elapsed.Seconds())
	c.logger.Infof("node %s: retrieved %d chunks in %v", node.Name(), len(chunks), elapsed)

	return nil
}

// checkNotFound asserts that an address that was never uploaded comes back as a
// not-found delivery rather than as data, an error, or silence. This exercises
// the path where retrieval exhausts every peer.
func (c *Check) checkNotFound(ctx context.Context, node *bee.Client, rnd *rand.Rand, o Options) error {
	// A freshly generated chunk that is never uploaded gives a valid address
	// that no node can hold.
	unknown, err := bee.NewRandomChunk(rnd, c.logger)
	if err != nil {
		return fmt.Errorf("create chunk: %w", err)
	}
	missing := unknown.Address()

	stream, err := node.API().ChunkStream.NewDownloadStream(ctx)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			c.logger.Debugf("node %s: closing download stream: %v", node.Name(), err)
		}
	}()

	if err := stream.Request([]swarm.Address{missing}, o.RequestTimeout); err != nil {
		return err
	}

	delivery, err := stream.Receive(o.RequestTimeout)
	if err != nil {
		return fmt.Errorf("address %s: %w", missing, err)
	}
	if !delivery.Address.Equal(missing) {
		return fmt.Errorf("%w: expected a delivery for %s, got one for %s", errChunkStream, missing, delivery.Address)
	}
	if delivery.Status != api.ChunkDeliveryNotFound {
		return fmt.Errorf("%w: address %s: expected status 0x%02x, got 0x%02x", errChunkStream, missing, api.ChunkDeliveryNotFound, delivery.Status)
	}

	c.metrics.NotFoundCounter.WithLabelValues(node.Name()).Inc()
	c.logger.Infof("node %s: unknown chunk correctly reported as not found", node.Name())

	return nil
}

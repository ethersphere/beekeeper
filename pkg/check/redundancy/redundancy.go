package redundancy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethersphere/bee/v2/pkg/file/pipeline/builder"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/storage"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

type Options struct {
	DataSize     int64
	PostageDepth uint64
	PostageLabel string
	PostageTTL   time.Duration
	Seed         int64
}

func NewDefaultOptions() Options {
	return Options{
		DataSize:     307200,
		PostageDepth: 22,
		PostageLabel: "test-label",
		PostageTTL:   24 * time.Hour,
		Seed:         time.Now().UnixNano(),
	}
}

var _ beekeeper.Action = (*Check)(nil)

type pipelineFn func(ctx context.Context, r io.Reader) (swarm.Address, error)

func requestPipelineFn(s storage.Putter, encrypt bool, rLevel redundancy.Level) pipelineFn {
	return func(ctx context.Context, r io.Reader) (swarm.Address, error) {
		pipe := builder.NewPipelineBuilder(ctx, s, encrypt, rLevel)
		return builder.FeedPipeline(ctx, pipe, r)
	}
}

type Check struct {
	logger logging.Logger
}

func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o any) (err error) {
	opts, ok := o.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	time.Sleep(10 * time.Second)

	for i := 1; i < 5; i++ { // skip level 0
		c.logger.Infof("started rLevel %d", i)
		uploadClient, downloadClient, err := getClients(ctx, cluster, opts.Seed)
		if err != nil {
			return fmt.Errorf("get clients: %w", err)
		}

		root, data, chunks, err := c.generateChunks(ctx, opts.DataSize, redundancy.Level(i))
		if err != nil {
			return fmt.Errorf("get chunks: %w", err)
		}
		c.logger.Infof("root hash: %s, chunks: %d", root.String(), len(chunks))

		batchID, err := uploadClient.GetOrCreateMutableBatch(ctx, opts.PostageTTL, opts.PostageDepth, opts.PostageLabel)
		if err != nil {
			return fmt.Errorf("get or create batch: %w", err)
		}

		// happy path
		err = c.uploadChunks(ctx, uploadClient, chunks, redundancy.Level(i), batchID, true)
		if err != nil {
			return fmt.Errorf("upload chunks: %w", err)
		}
		c.logger.Infof("upload completed. Downloading %s", root.String())
		fallbackMode := true
		d, err := downloadClient.DownloadBytes(ctx, root, &api.DownloadOptions{RedundancyFallbackMode: &fallbackMode})
		if err != nil {
			return fmt.Errorf("download bytes: %w", err)
		}

		if !bytes.Equal(data, d) {
			return fmt.Errorf("download and initial content dont match")
		}

		// non-happy path
		root, data, chunks, err = c.generateChunks(ctx, opts.DataSize, redundancy.Level(i))
		if err != nil {
			return fmt.Errorf("get chunks: %w", err)
		}
		c.logger.Infof("root hash: %s, chunks: %d", root.String(), len(chunks))
		err = c.uploadChunks(ctx, uploadClient, chunks, redundancy.Level(i), batchID, false)
		if err != nil {
			return fmt.Errorf("upload chunks: %w", err)
		}
		c.logger.Infof("upload completed. Downloading %s", root.String())
		d, _ = downloadClient.DownloadBytes(ctx, root, &api.DownloadOptions{RedundancyFallbackMode: &fallbackMode})
		if bytes.Equal(data, d) {
			return fmt.Errorf("download and initial content should not match")
		}

		c.logger.Infof("rLevel %d completed successfully", i)
	}
	return nil
}

func (c *Check) generateChunks(ctx context.Context, size int64, rLevel redundancy.Level) (swarm.Address, []byte, []swarm.Chunk, error) {
	putter := &splitPutter{
		chunks: make([]swarm.Chunk, 0),
	}

	buf := make([]byte, size)
	rnd := random.PseudoGenerator(time.Now().UnixNano())
	_, err := rnd.Read(buf)
	if err != nil {
		return swarm.ZeroAddress, nil, nil, err
	}

	p := requestPipelineFn(putter, false, rLevel)
	rootAddr, err := p(ctx, bytes.NewReader(buf))
	if err != nil {
		return swarm.ZeroAddress, nil, nil, err
	}

	return rootAddr, buf, putter.chunks, nil
}

func (c *Check) uploadChunks(ctx context.Context, client *bee.Client, chunks []swarm.Chunk, rLevel redundancy.Level, batchID string, shouldDownload bool) error {
	rate := 0.0
	if shouldDownload {
		switch rLevel {
		case redundancy.MEDIUM:
			rate = 0.01
		case redundancy.STRONG:
			rate = 0.05
		case redundancy.INSANE:
			rate = 0.1
		case redundancy.PARANOID:
			rate = 0.5
		}
	} else {
		switch rLevel {
		case redundancy.MEDIUM:
			rate = 0.2
		case redundancy.STRONG:
			rate = 0.25
		case redundancy.INSANE:
			rate = 0.35
		case redundancy.PARANOID:
			rate = 0.9
		}
	}

	rnd := random.PseudoGenerator(time.Now().UnixNano())
	indices := rnd.Perm(len(chunks) - 1)
	offset := int(rate * float64(len(chunks)))
	indices = append(indices[offset:], len(chunks)-1)

	c.logger.Infof("uploading %d chunks out of %d", len(indices), len(chunks))
	for i, j := range indices {
		_, err := client.UploadChunk(ctx, chunks[j].Data(), api.UploadOptions{
			BatchID: batchID,
			Direct:  true,
		})
		if err != nil {
			return fmt.Errorf("upload chunk %d of %d: %w", i+1, len(indices), err)
		}
	}
	return nil
}

func getClients(ctx context.Context, cluster orchestration.Cluster, seed int64) (*bee.Client, *bee.Client, error) {
	rnd := random.PseudoGenerator(seed)
	clients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return nil, nil, fmt.Errorf("get shuffled full node clients: %w", err)
	}

	if len(clients) < 2 {
		return nil, nil, fmt.Errorf("not enough full node clients in the cluster")
	}

	return clients[0], clients[1], nil
}

type splitPutter struct {
	chunks []swarm.Chunk
}

func (s *splitPutter) Put(_ context.Context, chunk swarm.Chunk) error {
	s.chunks = append(s.chunks, chunk)
	return nil
}

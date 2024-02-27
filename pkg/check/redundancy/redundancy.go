package redundancy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethersphere/bee/pkg/file/pipeline/builder"
	"github.com/ethersphere/bee/pkg/file/redundancy"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

type Options struct {
	PostageAmount int64
	PostageDepth  uint64
	Seed          int64
	DataSize      int64
}

func NewDefaultOptions() Options {
	return Options{
		PostageAmount: 1500000,
		PostageDepth:  22,
		Seed:          time.Now().UnixNano(),
		DataSize:      307200,
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

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o interface{}) (err error) {
	opts, ok := o.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	time.Sleep(10 * time.Second)

	for i := 0; i < 5; i++ {
		c.logger.Infof("started rLevel %d", i)
		uploadClient, downloadClient, err := getClients(ctx, cluster, opts.Seed)
		if err != nil {
			return fmt.Errorf("get clients: %w", err)
		}

		root, data, chunks, err := c.generateChunks(ctx, opts.DataSize, redundancy.Level(i), opts.Seed)
		if err != nil {
			return fmt.Errorf("get chunks: %w", err)
		}
		c.logger.Infof("root hash: %s, chunks: %d", root.String(), len(chunks))

		batchID, err := uploadClient.GetOrCreateBatch(ctx, opts.PostageAmount, opts.PostageDepth, "ci-redundancy")
		if err != nil {
			return fmt.Errorf("get or create batch: %w", err)
		}

		err = c.uploadChunks(ctx, uploadClient, chunks, redundancy.Level(i), opts.Seed, batchID)
		if err != nil {
			return fmt.Errorf("upload chunks: %w", err)
		}

		d, err := downloadClient.DownloadBytes(ctx, root)
		if err != nil {
			return fmt.Errorf("download bytes: %w", err)
		}

		if bytes.Compare(data, d) != 0 {
			return fmt.Errorf("download and initial content dont match")
		}

		c.logger.Infof("rLevel %d completed successfully", i)
	}
	return nil
}

func (c *Check) generateChunks(ctx context.Context, size int64, rLevel redundancy.Level, seed int64) (swarm.Address, []byte, []swarm.Chunk, error) {
	putter := &splitPutter{
		chunks: make([]swarm.Chunk, 0),
	}

	buf := make([]byte, size)
	rnd := random.PseudoGenerator(seed)
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

func (c *Check) uploadChunks(ctx context.Context, client *bee.Client, chunks []swarm.Chunk, rLevel redundancy.Level, seed int64, batchID string) error {
	rate := 0.0
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

	rnd := random.PseudoGenerator(seed)
	indices := rnd.Perm(len(chunks) - 1)
	offset := int(rate * float64(len(chunks)))
	indices = append(indices[offset:], len(chunks)-1)

	c.logger.Infof("uploading %d chunks out of %d", len(indices), len(chunks))
	for _, i := range indices {
		ref, err := client.UploadChunk(ctx, chunks[i].Data(), api.UploadOptions{
			BatchID: batchID,
			Direct:  true,
		})
		if err != nil {
			return err
		}
		c.logger.Infof("uploaded chunk: %s", ref.String())
	}
	return nil
}

func getClients(ctx context.Context, cluster orchestration.Cluster, seed int64) (*bee.Client, *bee.Client, error) {
	nodeNames := cluster.NodeNames()
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return nil, nil, err
	}
	rnd := random.PseudoGenerator(seed)
	var cUpload, cDownload *bee.Client
	for {
		perm := rnd.Perm(len(nodeNames))
		if perm[0] != perm[1] {
			cUpload = clients[nodeNames[perm[0]]]
			cDownload = clients[nodeNames[perm[1]]]
			break
		}
	}
	return cUpload, cDownload, nil
}

type splitPutter struct {
	chunks []swarm.Chunk
}

func (s *splitPutter) Put(_ context.Context, chunk swarm.Chunk) error {
	s.chunks = append(s.chunks, chunk)
	return nil
}

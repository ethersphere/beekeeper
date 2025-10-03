package test

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

type ChunkUploader struct {
	ctx     context.Context
	rnd     *rand.Rand
	name    string
	client  *bee.Client
	batchID string
	Overlay string
	logger  logging.Logger
}

func (c *ChunkUploader) Name() string {
	return c.name
}

func (cu *ChunkUploader) UploadRandomChunk() (*chunkV2, error) {
	chunk, err := bee.NewRandomChunk(cu.rnd, cu.logger)
	if err != nil {
		return nil, fmt.Errorf("node %s: %w", cu.name, err)
	}

	ref, err := cu.client.UploadChunk(cu.ctx, chunk.Data(), api.UploadOptions{BatchID: cu.batchID})
	if err != nil {
		return nil, fmt.Errorf("node %s: %w", cu.name, err)
	}

	return &chunkV2{
		addr: ref,
		data: chunk.Data(),
	}, nil
}

package bee

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
)

type BeeV2 struct {
	opts    CaseOptions
	name    string
	Addr    swarm.Address
	rnd     *rand.Rand
	overlay swarm.Address
	client  *bee.Client
}

func (b *BeeV2) Name() string {
	return b.name
}

func (b *BeeV2) Restricted() bool {
	return b.client.Config().Restricted
}

func (b *BeeV2) DownloadChunk(ctx context.Context, ref swarm.Address) ([]byte, error) {
	return b.client.DownloadChunk(ctx, ref, "")
}

// NewRandomFile returns new pseudorandom file
func (b *BeeV2) UploadRandomFile(ctx context.Context) (File, error) {
	name := fmt.Sprintf("%s-%s", b.opts.FileName, b.name)
	file := File{
		name: name,
		rand: b.rnd,
		size: b.opts.FileSize,
	}

	return file, b.UploadFile(ctx, file)
}

func (b *BeeV2) UploadFile(ctx context.Context, file File) error {
	depth := 2 + bee.EstimatePostageBatchDepth(b.opts.FileSize)
	batchID, err := b.client.CreatePostageBatch(ctx, b.opts.PostageAmount, depth, b.opts.GasPrice, b.opts.PostageLabel, false)
	if err != nil {
		return fmt.Errorf("node %s: created batch id %w", b.name, err)
	}
	fmt.Printf("node %s: created batch id %s\n", b.name, batchID)

	randomFile := bee.NewRandomFile(file.rand, file.name, file.size)
	if err := b.client.UploadFile(ctx, &randomFile, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %s: %w", b.name, err)
	}

	fmt.Println("Uploaded file to", b.name)

	return nil
}

func (b *BeeV2) ExpectToHaveFile(ctx context.Context, file File) error {
	size, hash, err := b.client.DownloadFile(ctx, file.address)
	if err != nil {
		return fmt.Errorf("node %s: %w", b.name, err)
	}

	fmt.Println("Downloaded file from", b.name)

	if !bytes.Equal(file.hash, hash) {
		return fmt.Errorf("file %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.address.String(), b.name, file.size, size)
	}

	return nil
}

func (b *BeeV2) NewChunkUploader(ctx context.Context) (*ChunkUploader, error) {
	o := b.opts
	batchID, err := b.client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
	if err != nil {
		return nil, fmt.Errorf("node %s: batch id %w", b.name, err)
	}
	fmt.Printf("node %s: batch id %s\n", b.name, batchID)

	return &ChunkUploader{
		ctx:     ctx,
		rnd:     b.rnd,
		name:    b.name,
		client:  b.client,
		batchID: batchID,
	}, nil
}

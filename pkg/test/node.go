package bee

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

type BeeV2 struct {
	opts    CaseOptions
	name    string
	Addr    swarm.Address
	rnd     *rand.Rand
	overlay swarm.Address
	client  *bee.Client
	logger  logging.Logger
}

func (b *BeeV2) Name() string {
	return b.name
}

func (b *BeeV2) Restricted() bool {
	return b.client.Config().Restricted
}

func (b *BeeV2) DownloadChunk(ctx context.Context, ref swarm.Address) ([]byte, error) {
	return b.client.DownloadChunk(ctx, ref, "", nil)
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
	batchID, err := b.client.CreatePostageBatch(ctx, b.opts.PostageAmount, depth, b.opts.PostageLabel, false)
	if err != nil {
		return fmt.Errorf("node %s: created batch id %w", b.name, err)
	}
	b.logger.Infof("node %s: created batch id %s", b.name, batchID)

	randomFile := bee.NewRandomFile(file.rand, file.name, file.size)
	if err := b.client.UploadFile(ctx, &randomFile, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %s: %w", b.name, err)
	}

	b.logger.Info("Uploaded file to", b.name)

	return nil
}

func (b *BeeV2) ExpectToHaveFile(ctx context.Context, file File) error {
	size, hash, err := b.client.DownloadFile(ctx, file.address, nil)
	if err != nil {
		return fmt.Errorf("node %s: %w", b.name, err)
	}

	b.logger.Info("Downloaded file from", b.name)

	if !bytes.Equal(file.hash, hash) {
		return fmt.Errorf("file %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.address.String(), b.name, file.size, size)
	}

	return nil
}

func (b *BeeV2) NewChunkUploader(ctx context.Context) (*ChunkUploader, error) {
	o := b.opts
	batchID, err := b.client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return nil, fmt.Errorf("node %s: batch id %w", b.name, err)
	}
	b.logger.Infof("node %s: batch id %s", b.name, batchID)

	return &ChunkUploader{
		ctx:     ctx,
		rnd:     b.rnd,
		name:    b.name,
		client:  b.client,
		batchID: batchID,
		logger:  b.logger,
	}, nil
}

type Wallet struct {
	BZZ, Native *big.Int
}

func (b *BeeV2) Withdraw(ctx context.Context, token, addr string) error {
	before, err := b.client.WalletBalance(ctx, token)
	if err != nil {
		return fmt.Errorf("(%s) wallet balance %w", b.name, err)
	}

	if err := b.client.Withdraw(ctx, token, addr); err != nil {
		return fmt.Errorf("(%s) withdraw balance %w", b.name, err)
	}

	time.Sleep(3 * time.Second)

	after, err := b.client.WalletBalance(ctx, token)
	if err != nil {
		return fmt.Errorf("(%s) wallet balance %w", b.name, err)
	}

	want := big.NewInt(0).Sub(before, big.NewInt(1000000))

	if after.Cmp(want) > 0 {
		return fmt.Errorf("incorrect balance after withdraw:\ngot  %d\nwant %d", after, want)
	}

	return nil
}

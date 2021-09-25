package bee

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
)

type beeV2 struct {
	o       CaseOptions
	name    string
	Addr    swarm.Address
	rnd     *rand.Rand
	overlay swarm.Address
	client  *bee.Client
}

func (n *beeV2) Name() string {
	return n.name
}

func (n *beeV2) DownloadChunk(ctx context.Context, ref swarm.Address) ([]byte, error) {
	return n.client.DownloadChunk(ctx, ref, "")
}

// NewRandomFile returns new pseudorandom file
func (node *beeV2) UploadRandomFile(ctx context.Context) (File, error) {
	name := fmt.Sprintf("%s-%s", node.o.FileName, node.name)
	file := File{
		name: name,
		rand: node.rnd,
		size: node.o.FileSize,
	}

	return file, node.UploadFile(ctx, file)
}

func (n beeV2) UploadFile(ctx context.Context, file File) error {
	depth := 2 + bee.EstimatePostageBatchDepth(n.o.FileSize)
	batchID, err := n.client.CreatePostageBatch(ctx, n.o.PostageAmount, depth, n.o.GasPrice, n.o.PostageLabel, false)
	if err != nil {
		return fmt.Errorf("node %s: created batch id %w", n.name, err)
	}
	fmt.Printf("node %s: created batch id %s\n", n.name, batchID)
	time.Sleep(n.o.PostageWait)

	randomFile := bee.NewRandomFile(file.rand, file.name, file.size)
	if err := n.client.UploadFile(ctx, &randomFile, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %s: %w", n.name, err)
	}

	fmt.Println("Uploaded file to", n.name)

	return nil
}

func (n beeV2) ExpectToHaveFile(ctx context.Context, file File) error {
	size, hash, err := n.client.DownloadFile(ctx, file.address)
	if err != nil {
		return fmt.Errorf("node %s: %w", n.name, err)
	}

	fmt.Println("Downloaded file from", n.name)

	if !bytes.Equal(file.hash, hash) {
		return fmt.Errorf("file %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.address.String(), n.name, file.size, size)
	}

	return nil
}

func (n *beeV2) NewChunkUploader(ctx context.Context) (*ChunkUploader, error) {
	o := n.o
	batchID, err := n.client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
	if err != nil {
		return nil, fmt.Errorf("node %s: batch id %w", n.name, err)
	}
	fmt.Printf("node %s: batch id %s\n", n.name, batchID)

	return &ChunkUploader{
		ctx:     ctx,
		rnd:     n.rnd,
		name:    n.name,
		client:  n.client,
		batchID: batchID,
	}, nil
}

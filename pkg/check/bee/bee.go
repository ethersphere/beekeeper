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
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

type CheckCase struct {
	ctx      context.Context
	clients  map[string]*bee.Client
	nodes    []nodeV2
	cluster  *orchestration.Cluster
	overlays orchestration.ClusterOverlays

	o               CaseOptions
	balancesHistory []orchestration.NodeGroupBalances
	rnd             *rand.Rand
}

type CaseOptions struct {
	FileName      string
	FileSize      int64
	GasPrice      string
	PostageAmount int64
	PostageLabel  string
	PostageWait   time.Duration
	Seed          int64
	PostageDepth  uint64
}

func NewCheckCase(ctx context.Context, cluster *orchestration.Cluster, o CaseOptions) (*CheckCase, error) {
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return nil, err
	}

	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return nil, err
	}

	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	flatOverlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return nil, err
	}

	rnds := random.PseudoGenerators(o.Seed, len(flatOverlays))
	fmt.Printf("Seed: %d\n", o.Seed)

	var (
		nodes []nodeV2
		count int
	)
	for name, addr := range flatOverlays {
		nodes = append(nodes, nodeV2{
			name:   name,
			Addr:   addr,
			client: clients[name],
			rnd:    rnds[count],
		})
		count++
	}

	return &CheckCase{
		ctx:      ctx,
		clients:  clients,
		overlays: overlays,
		nodes:    nodes,
		rnd:      rnd,
	}, nil
}

// NewRandomFile returns new pseudorandom file
func (node *nodeV2) UploadRandomFile(ctx context.Context) (FileV2, error) {
	name := fmt.Sprintf("%s-%s", node.o.FileName, node.name)
	file := FileV2{
		name: name,
		rand: node.rnd,
		size: node.o.FileSize,
	}

	return file, node.UploadFile(ctx, file)
}

func (c *CheckCase) SaveBalances() error {
	balances, err := c.cluster.Balances(c.ctx)

	if err != nil {
		return err
	}

	flatBalances := flattenBalances(balances)

	c.balancesHistory = append(c.balancesHistory, flatBalances)

	return nil
}

type nodeV2 struct {
	o       CaseOptions
	name    string
	Addr    swarm.Address
	rnd     *rand.Rand
	overlay swarm.Address
	client  *bee.Client
}

func (n *nodeV2) Name() string {
	return n.name
}

func (c *CheckCase) RandomNode() *nodeV2 {
	_, nodeName, overlay := c.overlays.Random(c.rnd)

	return &nodeV2{
		o:       c.o,
		name:    nodeName,
		overlay: overlay,
		client:  c.clients[nodeName],
	}
}

func (n nodeV2) UploadFile(ctx context.Context, file FileV2) error {
	depth := 2 + bee.EstimatePostageBatchDepth(n.o.FileSize)
	batchID, err := n.client.CreatePostageBatch(ctx, n.o.PostageAmount, depth, n.o.GasPrice, n.o.PostageLabel, false)
	if err != nil {
		return fmt.Errorf("node %s: created batch id %w", n.name, err)
	}
	fmt.Printf("node %s: created batch id %s\n", n.name, batchID)
	time.Sleep(n.o.PostageWait)

	filev1 := bee.NewRandomFile(file.rand, file.name, file.size)
	if err := n.client.UploadFile(ctx, &filev1, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %s: %w", n.name, err)
	}

	fmt.Println("Uploaded file to", n.name)

	return nil
}

func (n nodeV2) ExpectToHaveFile(ctx context.Context, file FileV2) error {
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

func (c *CheckCase) prevBalances() orchestration.NodeGroupBalances {
	len := len(c.balancesHistory)
	return c.balancesHistory[len-1]
}

type FileV2 struct {
	address swarm.Address
	name    string
	hash    []byte
	rand    *rand.Rand
	size    int64
}

func (c *CheckCase) LastNode() *nodeV2 {
	return &c.nodes[len(c.nodes)-1]
}

func (c *CheckCase) Node(index int) *nodeV2 {
	return &c.nodes[index]
}

type ChunkUploader struct {
	ctx     context.Context
	rnd     *rand.Rand
	name    string
	client  *bee.Client
	batchID string
	Overlay string
}

func (c *ChunkUploader) Name() string {
	return c.name
}

func (n *nodeV2) NewChunkUploader(ctx context.Context) (*ChunkUploader, error) {
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

func (cu *ChunkUploader) UploadRandomChunk() (*chunkV2, error) {
	chunk, err := bee.NewRandomChunk(cu.rnd)
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

type chunkV2 struct {
	addr swarm.Address
	data []byte
}

func (c *chunkV2) Addr() swarm.Address {
	return c.addr
}

func (c *chunkV2) AddrString() string {
	return c.addr.String()
}

func (c *chunkV2) Equals(data []byte) bool {
	return bytes.Equal(c.data, data)
}

func (c *chunkV2) Contains(data []byte) bool {
	return bytes.Contains(c.data, data)
}

func (c *chunkV2) Size() int {
	return len(c.data)
}

func (n *nodeV2) DownloadChunk(ctx context.Context, ref swarm.Address) ([]byte, error) {
	return n.client.DownloadChunk(ctx, ref, "")
}

package bee

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

type ClusterV2 struct {
	ctx             context.Context
	clients         map[string]*bee.Client
	nodes           []nodeV2
	o               ClusterOptions
	balances        func(ctx context.Context) (balances bee.ClusterBalances, err error)
	overlays        bee.ClusterOverlays
	balancesHistory []bee.NodeGroupBalances
	rnd             *rand.Rand
}

type ClusterOptions struct {
	DryRun             bool
	FileName           string
	FileSize           int64
	GasPrice           string
	PostageAmount      int64
	PostageLabel       string
	PostageWait        time.Duration
	Seed               int64
	UploadNodeCount    int
	WaitBeforeDownload time.Duration
	ChunksPerNode      int // number of chunks to upload per node
	PostageDepth       uint64
	MetricsPusher      *push.Pusher
}

func NewClusterV2(ctx context.Context, cluster *bee.Cluster, o ClusterOptions) (*ClusterV2, error) {
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
			Name:   name,
			Addr:   addr,
			client: clients[name],
			rnd:    rnds[count],
		})
		count++
	}

	return &ClusterV2{
		ctx:      ctx,
		clients:  clients,
		overlays: overlays,
		nodes:    nodes,
		rnd:      rnd,
		balances: func(ctx context.Context) (bee.ClusterBalances, error) {
			return cluster.Balances(ctx)
		},
	}, nil
}

// NewRandomFile returns new pseudorandom file
func (node *nodeV2) UploadRandomFile(ctx context.Context) (FileV2, error) {
	name := fmt.Sprintf("%s-%s", node.o.FileName, node.Name)
	file := FileV2{
		name: name,
		rand: node.rnd,
		size: node.o.FileSize,
	}

	return file, node.UploadFile(ctx, file)
}

func (c *ClusterV2) SaveBalances() error {
	balances, err := c.balances(c.ctx)

	if err != nil {
		return err
	}

	flatBalances := flattenBalances(balances)

	c.balancesHistory = append(c.balancesHistory, flatBalances)

	return nil
}

type nodeV2 struct {
	o       ClusterOptions
	Name    string
	Addr    swarm.Address
	rnd     *rand.Rand
	overlay swarm.Address
	client  *bee.Client
}

func (c *ClusterV2) RandomNode() *nodeV2 {
	_, nodeName, overlay := c.overlays.Random(c.rnd)

	return &nodeV2{
		o:       c.o,
		Name:    nodeName,
		overlay: overlay,
		client:  c.clients[nodeName],
	}
}

func (n nodeV2) UploadFile(ctx context.Context, file FileV2) error {
	depth := 2 + bee.EstimatePostageBatchDepth(n.o.FileSize)
	batchID, err := n.client.CreatePostageBatch(ctx, n.o.PostageAmount, depth, n.o.GasPrice, n.o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: created batch id %w", n.Name, err)
	}
	fmt.Printf("node %s: created batch id %s\n", n.Name, batchID)
	time.Sleep(n.o.PostageWait)

	filev1 := bee.NewRandomFile(file.rand, file.name, file.size)
	if err := n.client.UploadFile(ctx, &filev1, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %s: %w", n.Name, err)
	}

	fmt.Println("Uploaded file to", n.Name)

	return nil
}

func (n nodeV2) ExpectToHaveFile(ctx context.Context, file FileV2) error {
	size, hash, err := n.client.DownloadFile(ctx, file.address)
	if err != nil {
		return fmt.Errorf("node %s: %w", n.Name, err)
	}

	fmt.Println("Downloaded file from", n.Name)

	if !bytes.Equal(file.hash, hash) {
		return fmt.Errorf("file %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.address.String(), n.Name, file.size, size)
	}

	return nil
}

func (c *ClusterV2) prevBalances() bee.NodeGroupBalances {
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

func (c *ClusterV2) LastNode() *nodeV2 {
	return &c.nodes[len(c.nodes)-1]
}

func (c *ClusterV2) Node(index int) *nodeV2 {
	return &c.nodes[index]
}

type ChunkUploader struct {
	ctx     context.Context
	rnd     *rand.Rand
	Name    string
	client  *bee.Client
	batchID string
	Overlay string
}

func (n *nodeV2) NewChunkUploader(ctx context.Context, index int) (*ChunkUploader, error) {
	o := n.o
	batchID, err := n.client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
	if err != nil {
		return nil, fmt.Errorf("node %s: batch id %w", n.Name, err)
	}
	fmt.Printf("node %s: batch id %s\n", n.Name, batchID)

	return &ChunkUploader{
		ctx:     ctx,
		rnd:     n.rnd,
		Name:    n.Name,
		client:  n.client,
		batchID: batchID,
	}, nil
}

func (cu *ChunkUploader) UploadRandomChunk() (*chunkV2, error) {
	chunk, err := bee.NewRandomChunk(cu.rnd)
	if err != nil {
		return nil, fmt.Errorf("node %s: %w", cu.Name, err)
	}

	ref, err := cu.client.UploadChunk(cu.ctx, chunk.Data(), api.UploadOptions{BatchID: cu.batchID})
	if err != nil {
		return nil, fmt.Errorf("node %s: %w", cu.Name, err)
	}

	return &chunkV2{
		Addr: ref,
		data: chunk.Data(),
	}, nil
}

type chunkV2 struct {
	Addr swarm.Address
	data []byte
}

func (c *chunkV2) AddrString() string {
	return c.Addr.String()
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

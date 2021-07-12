package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents check options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	GasPrice        string
	MetricsPusher   *push.Pusher
	PostageAmount   int64
	PostageDepth    uint64
	PostageLabel    string
	PostageWait     time.Duration
	Seed            int64
	UploadNodeCount int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:   1,
		GasPrice:        "",
		MetricsPusher:   nil,
		PostageAmount:   1,
		PostageDepth:    16,
		PostageLabel:    "test-label",
		PostageWait:     5 * time.Second,
		Seed:            random.Int64(),
		UploadNodeCount: 1,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

var errRetrieval = errors.New("retrieval")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.MetricsPusher != nil {
		setUpMetrics(o)
	}

	cluster_v2, err := newClusterV2(ctx, cluster, o)
	if err != nil {
		return err
	}

	lastNode := cluster_v2.LastNode()

	for i := 0; i < o.UploadNodeCount; i++ {
		uploader, err := cluster_v2.Node(i).NewChunkUploader(ctx, i, o)
		if err != nil {
			return err
		}

		time.Sleep(o.PostageWait)

		for j := 0; j < o.ChunksPerNode; j++ {
			// time upload
			t0 := time.Now()

			chunk, err := uploader.uploadRandomChunk()

			if err != nil {
				return err
			}

			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(uploader.overlay).Inc()
			uploadTimeGauge.WithLabelValues(uploader.overlay, chunk.AddrString()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			// time download
			t1 := time.Now()

			data, err := lastNode.DownloadChunk(ctx, chunk.addr)

			if err != nil {
				return fmt.Errorf("node %s: %w", lastNode.name, err)
			}

			d1 := time.Since(t1)

			downloadedCounter.WithLabelValues(uploader.name).Inc()
			downloadTimeGauge.WithLabelValues(uploader.name, chunk.AddrString()).Set(d1.Seconds())
			downloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(chunk.Data(), data) {
				notRetrievedCounter.WithLabelValues(uploader.name).Inc()
				fmt.Printf("Node %s. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", lastNode.name, j, len(chunk.Data()), len(data), uploader.name, chunk.AddrString())
				if bytes.Contains(chunk.Data(), data) {
					fmt.Printf("Downloaded data is subset of the uploaded data\n")
				}
				return errRetrieval
			}

			retrievedCounter.WithLabelValues(uploader.name).Inc()
			fmt.Printf("Node %s. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", lastNode.name, j, uploader.name, chunk.AddrString())

			if o.MetricsPusher != nil {
				if err := o.MetricsPusher.Push(); err != nil {
					fmt.Printf("node %s: %v\n", lastNode.name, err)
				}
			}
		}
	}

	return
}

type clusterV2 struct {
	ctx     context.Context
	clients map[string]*bee.Client
	nodes   []nodeV2
}

func newClusterV2(ctx context.Context, cluster *bee.Cluster, o Options) (*clusterV2, error) {
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return nil, err
	}

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return nil, err
	}

	rnds := random.PseudoGenerators(o.Seed, len(overlays))
	fmt.Printf("Seed: %d\n", o.Seed)

	var (
		nodes []nodeV2
		count int
	)
	for name, addr := range overlays {
		nodes = append(nodes, nodeV2{
			name:    name,
			overlay: addr,
			client:  clients[name],
			rnd:     rnds[count],
		})
		count++
	}

	return &clusterV2{
		ctx:     ctx,
		clients: clients,
		nodes:   nodes,
	}, nil
}

func (c *clusterV2) LastNode() *nodeV2 {
	return &c.nodes[len(c.nodes)-1]
}

func (c *clusterV2) Node(index int) *nodeV2 {
	return &c.nodes[index]
}

type nodeV2 struct {
	name    string
	overlay swarm.Address
	client  *bee.Client
	rnd     *rand.Rand
}

func (n *nodeV2) DownloadChunk(ctx context.Context, ref swarm.Address) ([]byte, error) {
	return n.client.DownloadChunk(ctx, ref, "")
}

type chunkUploader struct {
	ctx     context.Context
	rnd     *rand.Rand
	name    string
	client  *bee.Client
	batchID string
	overlay string
}

func (n *nodeV2) NewChunkUploader(ctx context.Context, index int, o Options) (*chunkUploader, error) {
	batchID, err := n.client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
	if err != nil {
		return nil, fmt.Errorf("node %s: batch id %w", n.name, err)
	}
	fmt.Printf("node %s: batch id %s\n", n.name, batchID)

	return &chunkUploader{
		ctx:     ctx,
		rnd:     n.rnd,
		name:    n.name,
		client:  n.client,
		batchID: batchID,
	}, nil
}

func (cu *chunkUploader) uploadRandomChunk() (*chunkV2, error) {
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

func (c *chunkV2) AddrString() string {
	return c.addr.String()
}

func (c *chunkV2) Data() []byte {
	return c.data
}

func setUpMetrics(o Options) {
	o.MetricsPusher.Collector(uploadedCounter)
	o.MetricsPusher.Collector(uploadTimeGauge)
	o.MetricsPusher.Collector(uploadTimeHistogram)
	o.MetricsPusher.Collector(downloadedCounter)
	o.MetricsPusher.Collector(downloadTimeGauge)
	o.MetricsPusher.Collector(downloadTimeHistogram)
	o.MetricsPusher.Collector(retrievedCounter)
	o.MetricsPusher.Collector(notRetrievedCounter)
	o.MetricsPusher.Format(expfmt.FmtText)
}

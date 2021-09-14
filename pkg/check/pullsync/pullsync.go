package pullsync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents check options
type Options struct {
	ChunksPerNode              int // number of chunks to upload per node
	GasPrice                   string
	PostageAmount              int64
	PostageLabel               string
	PostageWait                time.Duration
	ReplicationFactorThreshold int // minimal replication factor per chunk
	Seed                       int64
	UploadNodeCount            int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:              1,
		GasPrice:                   "",
		PostageAmount:              1,
		PostageLabel:               "test-label",
		PostageWait:                5 * time.Second,
		ReplicationFactorThreshold: 2,
		Seed:                       random.Int64(),
		UploadNodeCount:            1,
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

var errPullSync = errors.New("pull sync")

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	var (
		rnds        = random.PseudoGenerators(o.Seed, o.UploadNodeCount)
		totalCopies = 0
	)

	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}
	uploader := clients["bee-0"]
	uploaderOverlay, err := uploader.Overlay(ctx)
	if err != nil {
		return fmt.Errorf("uploader overlay: %w", err)
	}
	start := time.Now()
	ch := bee.NewRandSwarmChunk(rnds[0])
	batchID, err := uploader.GetOrCreateBatch(ctx, 10000, 16, "", "")
	if err != nil {
		return fmt.Errorf("created batch id %w", err)
	}

	addr, err := uploader.UploadChunk(ctx, ch.Data(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	fmt.Printf("uploaded chunk %s to node %s\n", addr.String(), uploaderOverlay.String())
	sortedNodes := cluster.NodeNames()

	var wg sync.WaitGroup
	haves := make(map[string]struct{})
	var mtx sync.Mutex

	topCtx, _ := context.WithTimeout(ctx, 2*time.Minute)
LOOP:
	for {
		select {
		case <-topCtx.Done():
			break LOOP
		default:
		}

		wg.Add(len(sortedNodes))
		ctx2, _ := context.WithTimeout(ctx, 5*time.Second)
		for _, node := range sortedNodes {
			go func(nodeName string) {
				client := clients[nodeName]
				has, _ := client.HasChunk(ctx2, addr)
				if has {
					mtx.Lock()
					defer mtx.Unlock()
					if _, ok := haves[nodeName]; !ok {
						fmt.Printf("node %s (%s) has chunk %s, took %s\n", overlays[nodeName].String(), nodeName, addr.String(), time.Since(start))
						haves[nodeName] = struct{}{}
						totalCopies++
					}
				}
			}(node)
		}
		wg.Wait()
	}
	fmt.Printf("check ended, total copies: %d\n", totalCopies)
	return nil
}

// findName returns node name of a given swarm.Address in a given set of swarm.Addresses, or "" if not found
func findName(nodes map[string]swarm.Address, addr swarm.Address) (string, bool) {
	for n, a := range nodes {
		if addr.Equal(a) {
			return n, true
		}
	}

	return "", false
}

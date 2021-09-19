package chunkavailability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents check options
type Options struct {
	Seed int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		Seed: random.Int64(),
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{
		metrics: newMetrics(),
	}
}

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, metricsPusher *push.Pusher, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if metricsPusher != nil {
		mm.RegisterCollectors(c.Metrics()...)
	}

	var (
		rnds        = random.PseudoGenerator(o.Seed)
		totalCopies = 0
	)

	fmt.Printf("seed: %d\n", o.Seed)

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
	var ch swarm.Chunk
LOOP2:
	for {
		ch = bee.NewRandSwarmChunk(rnds[0])
		if swarm.Proximity(uploaderOverlay.Bytes(), ch.Address().Bytes()) > 3 {
			break LOOP2
		}
	}
	batchID, err := uploader.GetOrCreateBatch(ctx, 10000, 17, "", "")
	if err != nil {
		return fmt.Errorf("created batch id %w", err)
	}

	addr, err := uploader.UploadChunk(ctx, ch.Data(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	fmt.Printf("uploaded chunk %s to node %s, po %d\n", addr.String(), uploaderOverlay.String(), swarm.Proximity(uploaderOverlay.Bytes(), ch.Address().Bytes()))
	sortedNodes := cluster.NodeNames()

	var wg sync.WaitGroup
	haves := make(map[string]struct{})
	var mtx sync.Mutex

	topCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
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
				defer wg.Done()
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
		//fmt.Println("finished iteration", "nodes", len(sortedNodes))
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

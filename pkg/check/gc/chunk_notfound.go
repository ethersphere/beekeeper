package gc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	Seed             int64
	NodeGroup        string // TODO: support multi node group cluster
	StoreSize        int    // size of the node's localstore in chunks
	StoreSizeDivisor int    // divide store size by how much when uploading bytes
	Wait             int    // wait before check
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		NodeGroup:        "bee",
		Seed:             random.Int64(),
		StoreSize:        1000,
		StoreSizeDivisor: 3,
		Wait:             5,
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

// Run uploads a single chunk to a node, then uploads a lot of other chunks to see that it has been purged with gc
func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := cluster.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	pivot := rnd.Intn(ng.Size())
	pivotNode := sortedNodes[pivot]
	chunk, err := bee.NewRandomChunk(rnd)
	if err != nil {
		return err
	}

	ref, err := ng.NodeClient(pivotNode).UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
	if err != nil {
		return fmt.Errorf("node %s: %w", pivotNode, err)
	}
	fmt.Printf("uploaded chunk %s (%d bytes) to node %s: %s\n", ref.String(), len(chunk.Data()), pivotNode, overlays[pivotNode].String())

	b := make([]byte, (o.StoreSize/o.StoreSizeDivisor)*swarm.ChunkSize)

	for i := 0; i <= o.StoreSizeDivisor; i++ {
		_, err := rnd.Read(b)
		if err != nil {
			return fmt.Errorf("rand read: %w", err)
		}
		if _, err := ng.NodeClient(pivotNode).UploadBytes(ctx, b, api.UploadOptions{Pin: false}); err != nil {
			return fmt.Errorf("node %s: %w", pivotNode, err)
		}
		fmt.Printf("node %s: uploaded %d bytes.\n", pivotNode, len(b))
	}

	// allow time for syncing and GC
	time.Sleep(time.Duration(o.Wait) * time.Second)

	has, err := ng.NodeClient(pivotNode).HasChunk(ctx, ref)
	if err != nil {
		return fmt.Errorf("node has chunk: %w", err)
	}
	if has {
		return errors.New("expected chunk not found")
	}

	return nil
}

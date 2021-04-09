package localpinning

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// bytesFound uploads some bytes to a node, pins them, then uploads a lot of other chunks to see they are still there
func bytesFound(c *bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	size := (o.StoreSize / o.StoreSizeDivisor) * swarm.ChunkSize // size in bytes
	buf := make([]byte, size)
	_, err = rnd.Read(buf)
	if err != nil {
		return fmt.Errorf("rand buffer: %w", err)
	}

	addrs, err := addresses(buf)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	pivot := rnd.Intn(c.Size())
	pivotNode := sortedNodes[pivot]
	ref, err := ng.NodeClient(pivotNode).UploadBytes(ctx, buf, api.UploadOptions{Pin: true})
	if err != nil {
		return fmt.Errorf("node %s: upload bytes: %w", pivotNode, err)
	}

	fmt.Printf("uploaded and pinned %d bytes with hash %s to node %s: %s\n", size, ref.String(), pivotNode, overlays[pivotNode].String())

	for i := 0; i < o.StoreSizeDivisor; i++ {
		_, err := rnd.Read(buf)
		if err != nil {
			return fmt.Errorf("rand buffer: %w", err)
		}

		// upload without pinning
		a, err := ng.NodeClient(pivotNode).UploadBytes(ctx, buf, api.UploadOptions{Pin: false})
		if err != nil {
			return fmt.Errorf("node %s: upload bytes: %w", pivotNode, err)
		}

		fmt.Printf("uploaded %d unpinned bytes successfully, hash %s\n", size, a.String())
	}

	// allow the nodes to sync and do some GC
	time.Sleep(5 * time.Second)

	for _, a := range addrs {
		has, err := ng.NodeClient(pivotNode).HasChunk(ctx, a)
		if err != nil {
			return fmt.Errorf("node has chunk: %w", err)
		}
		if !has {
			return errors.New("pinning node: chunk not found")
		}
	}

	// cleanup
	for _, a := range addrs {
		if err := ng.NodeClient(pivotNode).UnpinChunk(ctx, a); err != nil {
			return fmt.Errorf("cannot unpin chunk: %w", err)
		}
	}

	return nil
}

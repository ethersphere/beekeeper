package localpinning

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// CheckRemoteChunksFound uploads some chunks to one node, pins them on another
// node (which does not have them locally), and then check that they are now
// available locally pinned on that node.
func CheckRemoteChunksFound(c *bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	for _, ng := range c.NodeGroups() {
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

		fmt.Printf("uploaded and pinned %d bytes (%d chunks) with hash %s to node %s: %s\n", size, len(addrs), ref.String(), pivotNode, overlays[pivotNode].String())

		// allow the nodes to sync and do some GC
		time.Sleep(15 * time.Second)

		for name := range overlays {
			if name == pivotNode {
				fmt.Printf("Node %s: pivot node, skipping it\n", name)
				continue
			}

			nodeClient := ng.NodeClient(name)

			fmt.Printf("Node %s: removing expected chunks\n", name)

			for _, a := range addrs {
				err := nodeClient.RemoveChunk(ctx, a)
				if err != nil {
					return fmt.Errorf("node has chunk: %w", err)
				}
			}

			chunksCountAfterRemove := 0

			for _, a := range addrs {
				has, err := nodeClient.HasChunk(ctx, a)
				if err != nil {
					return fmt.Errorf("node has chunk: %w", err)
				}
				if has {
					chunksCountAfterRemove++
				}
			}

			fmt.Printf("Node %s: has %d chunks after removal\n", name, chunksCountAfterRemove)

			fmt.Printf("Node %s: pinning chunks\n", name)

			err := nodeClient.PinBytes(ctx, ref)
			if err != nil {
				return fmt.Errorf("node %s: pin bytes: %w", name, err)
			}

			fmt.Printf("Node %s: checking for chunks\n", name)

			chunksCountAfterPin := 0

			for _, a := range addrs {
				has, err := nodeClient.HasChunk(ctx, a)
				if err != nil {
					return fmt.Errorf("node has chunk: %w", err)
				}
				if has {
					chunksCountAfterPin++
				}
			}

			if len(addrs) != chunksCountAfterPin {
				return fmt.Errorf("Node %s: has %d chunks (expected %d)", name, chunksCountAfterPin, len(addrs))
			}

			fmt.Printf("Node %s: has %d chunks (expected %d)\n", name, chunksCountAfterPin, len(addrs))
		}

		// cleanup
		for name, o := range overlays {
			fmt.Printf("Node %s: unpinning chunks\n", o.String())

			nodeClient := ng.NodeClient(name)

			for _, a := range addrs {
				err := nodeClient.UnpinChunk(ctx, a)
				if err != nil {
					return fmt.Errorf("cannot unpin chunk: %w", err)
				}
			}
		}

		for name, o := range overlays {
			fmt.Printf("Node %s: removing chunks\n", o.String())

			nodeClient := ng.NodeClient(name)

			for _, a := range addrs {
				err := nodeClient.RemoveChunk(ctx, a)
				if err != nil {
					return fmt.Errorf("cannot delete chunk: %w", err)
				}
			}
		}
	}

	return nil
}

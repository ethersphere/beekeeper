package localpinning

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// CheckBytesFound uploads some bytes to a node, pins them, then uploads a lot of other chunks to see they are still there
func CheckBytesFound(c bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	pivot := rnd.Intn(c.Size())
	size := (o.StoreSize / o.StoreSizeDivisor) * swarm.ChunkSize // size in bytes
	buf := make([]byte, size)
	_, err = rand.Read(buf)
	if err != nil {
		return fmt.Errorf("rand buffer: %w", err)
	}

	addrs, err := addresses(buf)
	if err != nil {
		return err
	}

	ref, err := c.Nodes[pivot].UploadBytes(ctx, buf, true)
	if err != nil {
		return fmt.Errorf("node %d: upload bytes: %w", pivot, err)
	}

	fmt.Printf("uploaded and pinned %d bytes with hash %s to node %d: %s\n", size, ref.String(), pivot, overlays[pivot].String())

	for i := 0; i <= o.StoreSizeDivisor+1; i++ {
		_, err := rand.Read(buf)
		if err != nil {
			return fmt.Errorf("rand buffer: %w", err)
		}

		// upload without pinning
		_, err = c.Nodes[pivot].UploadBytes(ctx, buf, false)
		if err != nil {
			return fmt.Errorf("node %d: upload bytes: %w", pivot, err)
		}
	}

	for _, a := range addrs {
		has, err := nodeHasChunk(ctx, &c.Nodes[pivot], a)
		if err != nil {
			return fmt.Errorf("node has chunk: %w", err)
		}
		if !has {
			return errors.New("pinning node: chunk not found")
		}
	}

	return nil
}

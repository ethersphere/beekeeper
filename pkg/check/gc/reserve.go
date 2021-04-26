package gc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

func CheckReserve(c *bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Println("gc: reserve check")
	fmt.Printf("Seed: %d\n", o.Seed)

	node := c.RandomNode(rnd)
	fmt.Printf("node %s\n", node.Name())

	adrs, err := node.Client().Addresses(ctx)
	if err != nil {
		return err
	}

	origState, err := node.Client().ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", origState)

	depth := capacityToDepth(origState.Radius, origState.Available)
	batch, err := node.Client().CreatePostageBatch(ctx, 1, depth, "test-label")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch with depth %d and amount %d\n", depth, 1)
	time.Sleep(time.Second * 5)

	state, err := node.Client().ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", state)

	lowValueChunks := chunkBatch(rnd, adrs.Overlay, int64(o.StoreSize), origState.Radius)
	for _, c := range lowValueChunks {
		_, err := node.Client().UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batch})
		if err != nil {
			return fmt.Errorf("low value chunk: %w", err)
		}
	}
	fmt.Printf("Uploaded %d chunks with batch depth %d\n", len(lowValueChunks), depth)

	// now create batch with higher value
	_, err = node.Client().CreatePostageBatch(ctx, 2, depth, "test-label")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch with depth %d and amount %d\n", depth, 2)
	time.Sleep(time.Second * 5)

	state, err = node.Client().ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}
	fmt.Println("reservestate:", state)

	hasCounter := 0
	for _, c := range lowValueChunks {
		has, _ := node.Client().HasChunk(ctx, c.Address())
		if has {
			hasCounter++
		}
	}

	fmt.Printf("retrieved low value chunks: %d, gc'd count: %d\n", hasCounter, len(lowValueChunks)-hasCounter)

	if len(lowValueChunks) == hasCounter {
		return errors.New("lowValueChunks was not gc'd")
	}

	return nil
}

func capacityToDepth(radius uint8, available int64) uint64 {
	depth := int(math.Log2(float64(available) * math.Exp2(float64(radius))))
	if depth < bee.MinimumBatchDepth {
		depth = bee.MinimumBatchDepth
	}

	return uint64(depth) + 1
}

func chunkBatch(rnd *rand.Rand, target swarm.Address, count int64, po uint8) []swarm.Chunk {

	ret := make([]swarm.Chunk, 0, count)

	for i := int64(0); i < count; i++ {
		ret = append(ret, bee.GenerateRandomChunkAt(rnd, target, po))
	}

	return ret
}

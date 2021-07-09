package gc

import (
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
)

// Options represents gc check options
type Options struct {
	CacheSize    int // size of the node's localstore in chunks
	GasPrice     string
	PostageLabel string
	PostageWait  time.Duration
	ReserveSize  int
	Seed         int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		CacheSize:    1000,
		GasPrice:     "500000000000",
		PostageLabel: "test-label",
		PostageWait:  50 * time.Second,
		ReserveSize:  1024,
		Seed:         0,
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

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(o.Seed)
	fmt.Println("gc: reserve check")
	fmt.Printf("Seed: %d\n", o.Seed)

	node, err := cluster.RandomNode(ctx, rnd)
	if err != nil {
		return fmt.Errorf("random node: %w", err)
	}
	fmt.Printf("chosen node %s\n", node.Name())

	const (
		loAmount = 1
		hiAmount = 3
		depth    = uint64(8)
	)

	var (
		client             = node.Client()
		pinnedChunk        swarm.Chunk
		higherRadiusChunks []swarm.Chunk
		lowValueChunks     []swarm.Chunk
	)

	addr, err := client.Addresses(ctx)
	if err != nil {
		return err
	}
	overlay := addr.Overlay

	origState, err := client.ReserveState(ctx)
	if err != nil {
		return fmt.Errorf("reservestate: %w", err)
	}

	fmt.Println("initial reserve state:", origState)

	for i, step := range []struct {
		desc        string
		buyStamp    bool
		stampAmount int64
		run         func(batchID string) error
	}{
		{
			desc:        "create low value stamp, upload pinned chunk and gc candidates",
			buyStamp:    true,
			stampAmount: loAmount,
			run: func(batchID string) error {
				pinnedChunk := bee.GenerateRandomChunkAt(rnd, overlay, 0)
				_, err = client.UploadChunk(ctx, pinnedChunk.Data(), api.UploadOptions{Pin: true, BatchID: batchID})
				if err != nil {
					return fmt.Errorf("unable to upload chunk: %w", err)
				}
				fmt.Printf("uploaded pinned chunk %q\n", pinnedChunk.Address())

				lowValueChunks = chunkBatch(rnd, overlay, 10, origState.Radius)
				for _, c := range lowValueChunks {
					_, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID})
					if err != nil {
						return fmt.Errorf("low value chunk: %w", err)
					}
				}
				fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(lowValueChunks), depth, loAmount, origState.Radius)
			},
		}, {
			desc:        "create second stamp and upload so that reserve and cache evictions are triggered",
			buyStamp:    true,
			stampAmount: hiAmount,
			run: func(batchID string) error {
				higherRadius := origState.Radius + 1

				higherRadiusChunks = chunkBatch(rnd, overlay, 10, higherRadius)
				for _, c := range higherRadiusChunks {
					if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: batchID}); err != nil {
						return fmt.Errorf("low value chunk: %w", err)
					}
				}
				fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(higherRadiusChunks), depth, loAmount, higherRadius)
			},
		}, {
			desc: "check evictions",
			run: func(_ string) error {
				hasCount := 0
				for _, c := range lowValueChunks {
					has, err := client.HasChunk(ctx, c.Address())
					if err != nil {
						return fmt.Errorf("low value chunk: %w", err)
					}
					if has {
						hasCount++
					}
				}

				lowValueChunksLen := len(lowValueChunks)
				wantCount := int(float64(lowValueChunksLen) * 0.9) // A 10% cache garbage collection is expected.
				fmt.Printf("retrieved low value chunks: %d, gc'd count: %d\n", hasCount, lowValueChunksLen-hasCount)

				if hasCount != wantCount {
					return fmt.Errorf("lowValueChunks gc count: has %d; want %d\n", hasCount, wantCount)
				}

				hasCount = 0
				for _, c := range higherRadiusChunks {
					has, err := client.HasChunk(ctx, c.Address())
					if err != nil {
						return fmt.Errorf("low value higher radius chunk: %w", err)
					}
					if has {
						hasCount++
					}
				}

				fmt.Printf("retrieved low value high radius chunks: %d, gc'd count: %d\n", hasCount, len(lowValueHigherRadiusChunks)-hasCount)

				if len(higherRadiusChunks) != hasCount {
					return fmt.Errorf("low value higher radius chunks were gc'd. Retrieved: %d, gc'd count: %d", hasCount, len(lowValueHigherRadiusChunks)-hasCount)
				}

				has, err := client.HasChunk(ctx, pinnedChunk.Address())
				if err != nil {
					return fmt.Errorf("unable to check pinned chunk %q: %w", pinnedChunk.Address(), err)
				}
				if !has {
					return fmt.Errorf("expected node pin for uploaded chunk %q", pinnedChunk.Address())
				}

			},
		}, {
			desc: "check pinning",
			run: func(_ string) error {
				pinned, err := client.GetPins(ctx)
				if err != nil {
					return fmt.Errorf("unable to get pins: %w", err)
				}
				if len(pinned) != 1 {
					return fmt.Errorf("unexpected pin count %d", len(pinned))
				}
				pinnedRef := pinned[0]

				if !pinnedChunk.Address().Equal(pinnedRef) {
					return fmt.Errorf("chunk %q is not pinned", pinnedChunk.Address())
				}

				if have, err := client.GetPinnedRootHash(ctx, pinnedRef); err != nil {
					return fmt.Errorf("unable to get pinned root hash: %w", err)
				} else if !have.Equal(pinnedRef) {
					return fmt.Errorf("address mismatch: have %q; want %q", have, pinnedRef)
				}

				if err := client.UnpinRootHash(ctx, pinnedRef); err != nil {
					return fmt.Errorf("cannot unpin chunk: %w", err)
				}

				if have, err := client.GetPinnedRootHash(ctx, pinnedRef); err != nil {
					return fmt.Errorf("unable to get pinned root hash: %w", err)
				} else if !have.Equal(swarm.ZeroAddress) {
					return fmt.Errorf("address mismatch: have %q; want none", have)
				}

				pinned, err = client.GetPins(ctx)
				if err != nil {
					return fmt.Errorf("unable to get pins: %w", err)
				}
				if len(pinned) > 0 {
					return errors.New("pin count is greater than zero")
				}
			},
		},
	} {
		fmt.Printf("executing step %d: %s", i, step.desc)
		var (
			batchID string
			err     error
		)

		if step.buyStamp {
			batchID, err = client.CreatePostageBatch(ctx, step.stampAmount, depth, o.GasPrice, o.PostageLabel)
			if err != nil {
				return fmt.Errorf("create batch: %w", err)
			}
			// allow the postage stamp to be picked up by the other nodes
			time.Sleep(o.PostageWait)
		}
		fmt.Printf("created batch id %s with depth %d and amount %d\n", batchID, depth, step.stampAmount)
		state, err = node.ReserveState(ctx)
		if err != nil {
			return fmt.Errorf("reservestate: %w", err)
		}
		fmt.Printf("%s reserve state: %s", statePrefix, state)

		if err := step.run(batchID, state); err != nil {
			return fmt.Errorf("step %d: run: %w", i, err)
		}
	}

	highValueChunks := chunkBatch(rnd, overlay, 10, state.Radius)
	for _, c := range highValueChunks {
		if _, err := client.UploadChunk(ctx, c.Data(), api.UploadOptions{BatchID: highValueBatch}); err != nil {
			return fmt.Errorf("high value chunks: %w", err)
		}
	}
	fmt.Printf("uploaded %d chunks with batch depth %d, amount %d, at radius %d\n", len(highValueChunks), depth, hiAmount, state.Radius)

	batchID, err = client.CreatePostageBatch(ctx, loAmount, depth-1, o.GasPrice, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("created batch id %s with depth %d and amount %d\n", batchID, depth, hiAmount)
	time.Sleep(o.PostageWait)

	if err = printReserveState(ctx, node.Client(), "post third batch buy"); err != nil {
		return fmt.Errorf("post first stamp: %w", err)
	}

	hasCount = 0
	for _, c := range highValueChunks {
		has, err := client.HasChunk(ctx, c.Address())
		if err != nil {
			return fmt.Errorf("high value chunk: %w", err)
		}
		if has {
			hasCount++
		}
	}

	fmt.Printf("retrieved high value chunks: %d, gc'd count: %d\n", hasCount, len(highValueChunks)-hasCount)

	if len(highValueChunks) != hasCount {
		return fmt.Errorf("high value chunks were gc'd. Retrieved: %d,  gc'd count: %d", hasCount, len(highValueChunks)-hasCount)
	}

	return nil
}

func chunkBatch(rnd *rand.Rand, target swarm.Address, count int, po uint8) []swarm.Chunk {
	start := time.Now()
	defer func() {
		fmt.Printf("chunkBatch took: %v\n", time.Since(start))
	}()
	chunks := make([]swarm.Chunk, count)
	for i := range chunks {
		chunks[i] = bee.GenerateRandomChunkAt(rnd, target, po)
	}
	return chunks
}

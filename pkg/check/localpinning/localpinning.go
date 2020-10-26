package localpinning

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/ethersphere/bee/pkg/file/pipeline/builder"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
)

// Options represents localpinning check options
type Options struct {
	StoreSize        int // size of the node's localstore in chunks
	StoreSizeDivisor int // divide store size by how much when uploading bytes
	Seed             int64
}

func nodeHasChunk(ctx context.Context, n *bee.Node, addr swarm.Address) (bool, error) {
	var counter = 0
	const retries = 5
	for i := 0; i < retries; i++ {
		time.Sleep(1 * time.Second)
		has, err := n.HasChunk(ctx, addr)
		if err != nil {
			if counter > 5 {
				return false, err
			}
			if errors.Is(err, api.ErrServiceUnavailable) {
				continue
			}
		}
		if has {
			return true, nil
		}
	}
	return false, nil
}

func addresses(buf []byte) ([]swarm.Address, error) {
	storer := &loggingStore{}
	ctx := context.Background()
	r := bytes.NewReader(buf)
	pipe := builder.NewPipelineBuilder(ctx, storer, storage.ModePutUpload, false)
	_, err := builder.FeedPipeline(ctx, pipe, r, int64(len(buf)))
	if err != nil {
		return nil, err
	}
	return storer.addrs, nil
}

type loggingStore struct {
	addrs []swarm.Address
}

func (l *loggingStore) Get(ctx context.Context, mode storage.ModeGet, addr swarm.Address) (ch swarm.Chunk, err error) {
	panic("not implemented")
}

func (l *loggingStore) Put(ctx context.Context, mode storage.ModePut, chs ...swarm.Chunk) (exist []bool, err error) {
	a := make([]swarm.Address, len(chs))
	for i, v := range chs {
		a[i] = v.Address()
	}

	l.addrs = append(l.addrs, a...)
	return nil, nil
}

func (l *loggingStore) GetMulti(ctx context.Context, mode storage.ModeGet, addrs ...swarm.Address) (ch []swarm.Chunk, err error) {
	panic("not implemented")
}

func (l *loggingStore) Has(ctx context.Context, addr swarm.Address) (yes bool, err error) {
	panic("not implemented")
}

func (l *loggingStore) HasMulti(ctx context.Context, addrs ...swarm.Address) (yes []bool, err error) {
	panic("not implemented")
}

func (l *loggingStore) Set(ctx context.Context, mode storage.ModeSet, addrs ...swarm.Address) (err error) {
	panic("not implemented")
}

func (l *loggingStore) LastPullSubscriptionBinID(bin uint8) (id uint64, err error) {
	panic("not implemented")
}

func (l *loggingStore) SubscribePull(ctx context.Context, bin uint8, since, until uint64) (<-chan storage.Descriptor, <-chan struct{}, func()) {
	panic("not implemented")
}

func (l *loggingStore) SubscribePush(ctx context.Context) (c <-chan swarm.Chunk, stop func()) {
	panic("not implemented")
}

func (l *loggingStore) PinnedChunks(ctx context.Context, cursor swarm.Address) (pinnedChunks []*storage.Pinner, err error) {
	panic("not implemented")
}

func (l *loggingStore) PinInfo(address swarm.Address) (uint64, error) {
	panic("not implemented")
}

func (l *loggingStore) Close() error {
	panic("not implemented")
}

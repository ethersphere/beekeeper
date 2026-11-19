package erasurecoding

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/ethersphere/bee/v2/pkg/bmt"
	"github.com/ethersphere/bee/v2/pkg/file"
	"github.com/ethersphere/bee/v2/pkg/file/pipeline/builder"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/storage"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

type chunkStore struct {
	mu     sync.Mutex
	chunks map[string]swarm.Chunk
}

func newChunkStore() *chunkStore {
	return &chunkStore{chunks: make(map[string]swarm.Chunk)}
}

func (s *chunkStore) Put(_ context.Context, ch swarm.Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chunks[ch.Address().ByteString()] = ch
	return nil
}

func (s *chunkStore) Get(_ context.Context, addr swarm.Address) (swarm.Chunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.chunks[addr.ByteString()]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return ch, nil
}

func (s *chunkStore) Without(drop []swarm.Address) *chunkStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	skip := make(map[string]struct{}, len(drop))
	for _, a := range drop {
		skip[a.ByteString()] = struct{}{}
	}
	out := newChunkStore()
	for k, ch := range s.chunks {
		if _, ok := skip[k]; !ok {
			out.chunks[k] = ch
		}
	}
	return out
}

func (s *chunkStore) Chunks() []swarm.Chunk {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]swarm.Chunk, 0, len(s.chunks))
	for _, ch := range s.chunks {
		out = append(out, ch)
	}
	return out
}

type node struct {
	Address  swarm.Address
	Data     []swarm.Address
	Parities []swarm.Address
	Leaf     bool
}

type tree struct {
	Root     swarm.Address
	Level    redundancy.Level
	Store    *chunkStore
	Nodes    []node
	Replicas []swarm.Address
}

func (t *tree) LeafNodes() []node {
	var out []node
	for _, n := range t.Nodes {
		if n.Leaf {
			out = append(out, n)
		}
	}
	return out
}

func split(ctx context.Context, data []byte, level redundancy.Level) (*tree, error) {
	if level == redundancy.NONE || !level.Validate() {
		return nil, fmt.Errorf("unsupported redundancy level %d", level)
	}
	if len(data) <= swarm.ChunkSize {
		return nil, fmt.Errorf("data size %d must exceed one chunk (%d bytes)", len(data), swarm.ChunkSize)
	}

	store := newChunkStore()
	pipe := builder.NewPipelineBuilder(ctx, store, false, level)
	root, err := builder.FeedPipeline(ctx, pipe, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("split: %w", err)
	}

	t := &tree{Root: root, Level: level, Store: store}
	if err := t.walk(root); err != nil {
		return nil, fmt.Errorf("index tree: %w", err)
	}

	for _, ch := range store.Chunks() {
		if !ch.Address().Equal(root) && soc.Valid(ch) {
			t.Replicas = append(t.Replicas, ch.Address())
		}
	}
	if len(t.Replicas) != level.GetReplicaCount() {
		return nil, fmt.Errorf("expected %d root replicas, got %d", level.GetReplicaCount(), len(t.Replicas))
	}

	return t, nil
}

func (t *tree) walk(addr swarm.Address) error {
	ch, err := t.Store.Get(context.Background(), addr)
	if err != nil {
		return fmt.Errorf("chunk %s: %w", addr, err)
	}
	level, span := chunkSpan(ch)
	if span <= swarm.ChunkSize {
		return nil
	}

	payload := ch.Data()[swarm.SpanSize:]
	size, err := file.ChunkPayloadSize(payload)
	if err != nil {
		return fmt.Errorf("chunk %s: %w", addr, err)
	}
	_, parities := file.ReferenceCount(span, level, false)
	addrs, shards := file.ChunkAddresses(payload[:size], parities, swarm.HashSize)

	n := node{
		Address:  addr,
		Data:     addrs[:shards],
		Parities: addrs[shards:],
		Leaf:     true,
	}
	for _, child := range n.Data {
		childCh, err := t.Store.Get(context.Background(), child)
		if err != nil {
			return fmt.Errorf("child %s of %s: %w", child, addr, err)
		}
		if _, childSpan := chunkSpan(childCh); childSpan > swarm.ChunkSize {
			n.Leaf = false
			if err := t.walk(child); err != nil {
				return err
			}
		}
	}
	for _, p := range n.Parities {
		if _, err := t.Store.Get(context.Background(), p); err != nil {
			return fmt.Errorf("parity %s of %s: %w", p, addr, err)
		}
	}

	t.Nodes = append(t.Nodes, n)
	return nil
}

func chunkSpan(ch swarm.Chunk) (redundancy.Level, uint64) {
	level, span := redundancy.DecodeSpan(ch.Data()[:swarm.SpanSize])
	return level, bmt.LengthFromSpan(span)
}

// drops returns count children of n to hold back, data shards first so that
// decoding is required, then parities once all data shards are taken.
func drops(n node, count int, rnd *rand.Rand) ([]swarm.Address, error) {
	if count > len(n.Data)+len(n.Parities) {
		return nil, fmt.Errorf("cannot drop %d of %d children", count, len(n.Data)+len(n.Parities))
	}
	out := make([]swarm.Address, 0, count)
	for _, i := range rnd.Perm(len(n.Data)) {
		if len(out) == count {
			return out, nil
		}
		out = append(out, n.Data[i])
	}
	for _, i := range rnd.Perm(len(n.Parities)) {
		if len(out) == count {
			break
		}
		out = append(out, n.Parities[i])
	}
	return out, nil
}

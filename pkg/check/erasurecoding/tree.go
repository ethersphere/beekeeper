package erasurecoding

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"

	"github.com/ethersphere/bee/v2/pkg/file"
	"github.com/ethersphere/bee/v2/pkg/file/pipeline/builder"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// Case is how many children of one intermediate chunk are withheld from the upload.
type Case int

const (
	// ParityOnly withholds every parity chunk, so all data chunks are still present.
	ParityOnly Case = iota
	// AtThreshold withholds as many children as there are parities, the most that can be
	// rebuilt: exactly as many children survive as there are data chunks.
	AtThreshold
	// OverThreshold withholds one child more, so fewer children survive than there are data
	// chunks and the group cannot be rebuilt.
	OverThreshold
)

// Cases lists every case in the order the check runs them.
var Cases = []Case{ParityOnly, AtThreshold, OverThreshold}

func (c Case) String() string {
	switch c {
	case ParityOnly:
		return "parity-only"
	case AtThreshold:
		return "at-threshold"
	case OverThreshold:
		return "over-threshold"
	default:
		return fmt.Sprintf("case(%d)", int(c))
	}
}

// ExpectDownload reports whether the download should succeed for the case. With the
// fallback off Bee fetches data chunks only, so any missing data chunk fails it.
func (c Case) ExpectDownload(fallback bool) bool {
	switch c {
	case ParityOnly:
		return true
	case AtThreshold:
		return fallback
	default:
		return false
	}
}

// Group is the children of one intermediate chunk: data references followed by parity references.
type Group struct {
	Depth      int  // distance from the root chunk, which is depth 0
	LeafParent bool // all data children are leaf data chunks
	Data       []swarm.Address
	Parity     []swarm.Address
}

// Drops picks the chunk addresses to withhold for the case. Any set of children is
// recoverable while at least len(Data) of the len(Data)+len(Parity) survive, so the
// threshold is len(Parity) withheld children. Data chunks are withheld first, so a data
// chunk is always missing and the download has to rebuild it; PARANOID has more parities
// than data chunks, so parities make up the rest there.
func (g Group) Drops(c Case, rnd *rand.Rand) []swarm.Address {
	switch c {
	case ParityOnly:
		return g.Parity
	case AtThreshold:
		return g.withhold(len(g.Parity), rnd)
	case OverThreshold:
		return g.withhold(len(g.Parity)+1, rnd)
	default:
		return nil
	}
}

// withhold picks n children at random, data chunks first.
func (g Group) withhold(n int, rnd *rand.Rand) []swarm.Address {
	nData := min(n, len(g.Data))

	picked := make([]swarm.Address, 0, n)
	for _, i := range rnd.Perm(len(g.Data))[:nData] {
		picked = append(picked, g.Data[i])
	}
	for _, i := range rnd.Perm(len(g.Parity))[:n-nData] {
		picked = append(picked, g.Parity[i])
	}

	return picked
}

// Tree is the chunk tree of one split blob.
type Tree struct {
	Root   swarm.Address
	Level  redundancy.Level
	Chunks []swarm.Chunk
	Groups []Group
}

// Split builds the chunk tree for data with Bee's splitter, keeping every chunk in memory.
func Split(ctx context.Context, data []byte, level redundancy.Level) (*Tree, error) {
	store := &memStore{chunks: make(map[string]swarm.Chunk)}

	pipe := builder.NewPipelineBuilder(ctx, store, false, level)
	root, err := builder.FeedPipeline(ctx, pipe, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("split data: %w", err)
	}

	groups, err := buildGroups(store.chunks, root, level)
	if err != nil {
		return nil, fmt.Errorf("read tree: %w", err)
	}

	return &Tree{Root: root, Level: level, Chunks: treeChunks(store.list, root, groups), Groups: groups}, nil
}

// treeChunks keeps only the chunks reachable from the root, in the order they were written.
// The splitter also writes signed replicas of the root chunk, which are not part of the
// tree and must not be uploaded as plain chunks.
func treeChunks(written []swarm.Chunk, root swarm.Address, groups []Group) []swarm.Chunk {
	inTree := map[string]bool{root.ByteString(): true}
	for _, g := range groups {
		for _, a := range g.Data {
			inTree[a.ByteString()] = true
		}
		for _, a := range g.Parity {
			inTree[a.ByteString()] = true
		}
	}

	chunks := make([]swarm.Chunk, 0, len(inTree))
	for _, ch := range written {
		if inTree[ch.Address().ByteString()] {
			chunks = append(chunks, ch)
		}
	}

	return chunks
}

// SelectGroups returns the groups to test: the first one whose children are leaf data
// chunks and, when all is set, the first one at every other depth too.
func (t *Tree) SelectGroups(all bool) []Group {
	var selected []Group

	seenDepth := make(map[int]bool)
	leafDone := false

	for _, g := range t.Groups {
		switch {
		case g.LeafParent && !leafDone:
			leafDone = true
			seenDepth[g.Depth] = true
			selected = append(selected, g)
		case all && !g.LeafParent && !seenDepth[g.Depth]:
			seenDepth[g.Depth] = true
			selected = append(selected, g)
		}
	}

	return selected
}

// memStore collects the chunks the splitter writes. The splitter calls Put from several
// goroutines, so access is locked.
type memStore struct {
	mu     sync.Mutex
	chunks map[string]swarm.Chunk
	list   []swarm.Chunk
}

func (s *memStore) Put(_ context.Context, ch swarm.Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chunks[ch.Address().ByteString()]; !ok {
		s.chunks[ch.Address().ByteString()] = ch
		s.list = append(s.list, ch)
	}

	return nil
}

// chunkSpan returns the data length a chunk covers; the level encoded in the span is dropped.
func chunkSpan(ch swarm.Chunk) uint64 {
	_, span := redundancy.DecodeSpan(ch.Data()[:swarm.SpanSize])

	return binary.LittleEndian.Uint64(span)
}

// buildGroups walks the tree from the root and returns one Group per intermediate chunk.
func buildGroups(chunks map[string]swarm.Chunk, root swarm.Address, level redundancy.Level) ([]Group, error) {
	type node struct {
		addr  swarm.Address
		depth int
	}

	var groups []Group

	queue := []node{{addr: root}}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]

		ch, ok := chunks[n.addr.ByteString()]
		if !ok {
			return nil, fmt.Errorf("chunk %s missing from the split output", n.addr)
		}

		span := chunkSpan(ch)
		if span <= swarm.ChunkSize {
			continue // data chunk
		}

		payload := ch.Data()[swarm.SpanSize:]

		size, err := file.ChunkPayloadSize(payload)
		if err != nil {
			return nil, fmt.Errorf("chunk %s: %w", n.addr, err)
		}

		dataCnt, parityCnt := file.ReferenceCount(span, level, false)
		addrs, shardCnt := file.ChunkAddresses(payload[:size], parityCnt, swarm.HashSize)

		if shardCnt != dataCnt || len(addrs) != dataCnt+parityCnt {
			return nil, fmt.Errorf("chunk %s: found %d data and %d parity references, want %d and %d",
				n.addr, shardCnt, len(addrs)-shardCnt, dataCnt, parityCnt)
		}

		g := Group{Depth: n.depth, LeafParent: true, Data: addrs[:shardCnt], Parity: addrs[shardCnt:]}

		for _, child := range g.Data {
			cc, ok := chunks[child.ByteString()]
			if !ok {
				return nil, fmt.Errorf("chunk %s missing from the split output", child)
			}
			if chunkSpan(cc) > swarm.ChunkSize {
				g.LeafParent = false
				queue = append(queue, node{addr: child, depth: n.depth + 1})
			}
		}

		groups = append(groups, g)
	}

	return groups, nil
}

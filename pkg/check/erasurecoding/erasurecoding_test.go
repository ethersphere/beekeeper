package erasurecoding_test

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/ethersphere/bee/v2/pkg/file/joiner"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy/getter"
	"github.com/ethersphere/bee/v2/pkg/storage"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/check/erasurecoding"
)

var levels = []redundancy.Level{redundancy.MEDIUM, redundancy.STRONG, redundancy.INSANE, redundancy.PARANOID}

func randomData(t *testing.T, size int) []byte {
	t.Helper()

	data := make([]byte, size)
	if _, err := rand.New(rand.NewSource(int64(size))).Read(data); err != nil {
		t.Fatal(err)
	}

	return data
}

// store serves the chunks of a tree except the withheld ones, like a network that never got them.
type store struct {
	chunks map[string]swarm.Chunk
}

func newStore(tree *erasurecoding.Tree, withheld []swarm.Address) *store {
	s := &store{chunks: make(map[string]swarm.Chunk)}
	for _, ch := range tree.Chunks {
		s.chunks[ch.Address().ByteString()] = ch
	}
	for _, a := range withheld {
		delete(s.chunks, a.ByteString())
	}

	return s
}

func (s *store) Get(_ context.Context, a swarm.Address) (swarm.Chunk, error) {
	if ch, ok := s.chunks[a.ByteString()]; ok {
		return ch, nil
	}

	return nil, storage.ErrNotFound
}

func (s *store) Put(context.Context, swarm.Chunk) error { return nil }

// join reads the blob back through Bee's joiner, the code path a node uses on download.
func join(t *testing.T, tree *erasurecoding.Tree, withheld []swarm.Address, fallback bool) ([]byte, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctx = getter.SetStrict(ctx, !fallback)

	s := newStore(tree, withheld)

	j, _, err := joiner.New(ctx, s, s, tree.Root, tree.Level)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(io.NewSectionReader(j, 0, j.Size()))
}

func groupChildren(tree *erasurecoding.Tree) int {
	n := 0
	for _, g := range tree.Groups {
		n += len(g.Data) + len(g.Parity)
	}

	return n
}

func TestSplit_groups(t *testing.T) {
	t.Parallel()

	for _, level := range levels {
		for _, size := range []int{300 * 1024, 1024 * 1024} {
			tree, err := erasurecoding.Split(context.Background(), randomData(t, size), level)
			if err != nil {
				t.Fatalf("level %d size %d: %v", level, size, err)
			}

			if len(tree.Groups) == 0 {
				t.Fatalf("level %d size %d: no intermediate chunks", level, size)
			}

			have := make(map[string]bool, len(tree.Chunks))
			for _, ch := range tree.Chunks {
				have[ch.Address().ByteString()] = true
			}

			for _, g := range tree.Groups {
				if len(g.Parity) == 0 {
					t.Errorf("level %d size %d: group at depth %d has no parities", level, size, g.Depth)
				}
				for _, a := range append(append([]swarm.Address{}, g.Data...), g.Parity...) {
					if !have[a.ByteString()] {
						t.Errorf("level %d size %d: child %s was not written by the splitter", level, size, a)
					}
				}
			}

			if want := 1 + groupChildren(tree); len(tree.Chunks) != want {
				t.Errorf("level %d size %d: %d chunks, want %d (root plus every child, no root replicas)", level, size, len(tree.Chunks), want)
			}

			if tree.Groups[0].Depth != 0 {
				t.Errorf("level %d size %d: first group depth = %d, want the root at 0", level, size, tree.Groups[0].Depth)
			}
		}
	}
}

func TestSplit_twoTreeLevels(t *testing.T) {
	t.Parallel()

	for _, level := range levels {
		tree, err := erasurecoding.Split(context.Background(), randomData(t, 1024*1024), level)
		if err != nil {
			t.Fatal(err)
		}

		if got := len(tree.SelectGroups(true)); got < 2 {
			t.Errorf("level %d: 1 MiB gives %d selectable groups, want at least 2 tree levels", level, got)
		}
	}
}

func TestGroupDrops_counts(t *testing.T) {
	t.Parallel()

	for _, level := range levels {
		tree, err := erasurecoding.Split(context.Background(), randomData(t, 1024*1024), level)
		if err != nil {
			t.Fatal(err)
		}

		for _, g := range tree.SelectGroups(true) {
			inGroup := make(map[string]bool)
			for _, a := range append(append([]swarm.Address{}, g.Data...), g.Parity...) {
				inGroup[a.ByteString()] = true
			}

			for _, c := range erasurecoding.Cases {
				drops := g.Drops(c, rand.New(rand.NewSource(1)))

				want := map[erasurecoding.Case]int{
					erasurecoding.ParityOnly:    len(g.Parity),
					erasurecoding.AtThreshold:   len(g.Parity),
					erasurecoding.OverThreshold: len(g.Parity) + 1,
				}[c]
				if len(drops) != want {
					t.Errorf("level %d depth %d %s: %d drops, want %d", level, g.Depth, c, len(drops), want)
				}

				seen := make(map[string]bool)
				for _, a := range drops {
					if a.Equal(tree.Root) {
						t.Errorf("level %d depth %d %s: root chunk is withheld", level, g.Depth, c)
					}
					if !inGroup[a.ByteString()] {
						t.Errorf("level %d depth %d %s: %s is not a child of the group", level, g.Depth, c, a)
					}
					if seen[a.ByteString()] {
						t.Errorf("level %d depth %d %s: %s is withheld twice", level, g.Depth, c, a)
					}
					seen[a.ByteString()] = true
				}
			}
		}
	}
}

// TestGroupDrops_matchBeeRecovery is the check's expectation table run against Bee's
// own joiner, with no network: it proves the thresholds before they are tried on a cluster.
func TestGroupDrops_matchBeeRecovery(t *testing.T) {
	t.Parallel()

	for _, level := range levels {
		data := randomData(t, 1024*1024)

		tree, err := erasurecoding.Split(context.Background(), data, level)
		if err != nil {
			t.Fatal(err)
		}

		for _, g := range tree.SelectGroups(true) {
			for _, c := range erasurecoding.Cases {
				drops := g.Drops(c, rand.New(rand.NewSource(7)))

				for _, fallback := range []bool{true, false} {
					got, err := join(t, tree, drops, fallback)
					gotOK := err == nil && bytes.Equal(got, data)

					if want := c.ExpectDownload(fallback); gotOK != want {
						t.Errorf("level %d depth %d %s fallback=%v: download ok = %v (err %v), want %v",
							level, g.Depth, c, fallback, gotOK, err, want)
					}
				}
			}
		}
	}
}

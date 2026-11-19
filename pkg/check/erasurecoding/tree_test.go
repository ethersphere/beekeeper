package erasurecoding_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/ethersphere/bee/v2/pkg/file/joiner"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy/getter"
	"github.com/ethersphere/bee/v2/pkg/log"
	"github.com/ethersphere/bee/v2/pkg/storage"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/check/erasurecoding"
	"github.com/ethersphere/beekeeper/pkg/random"
)

var levels = []redundancy.Level{redundancy.MEDIUM, redundancy.STRONG, redundancy.INSANE, redundancy.PARANOID}

func randomData(t testing.TB, size int, seed int64) []byte {
	t.Helper()
	buf := make([]byte, size)
	if _, err := random.PseudoGenerator(seed).Read(buf); err != nil {
		t.Fatal(err)
	}
	return buf
}

func split(t testing.TB, size int, level redundancy.Level) ([]byte, *erasurecoding.Tree) {
	t.Helper()
	data := randomData(t, size, int64(size)+int64(level))
	tr, err := erasurecoding.Split(context.Background(), data, level)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	return data, tr
}

func join(t *testing.T, store storage.Getter, root swarm.Address, level redundancy.Level) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fallback := true
	timeout := "100ms"
	ctx, err := getter.SetConfigInContext(ctx, nil, &fallback, &timeout, log.Noop)
	if err != nil {
		t.Fatal(err)
	}
	j, _, err := joiner.New(ctx, store, nopPutter{}, root, level)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(j)
}

type nopPutter struct{}

func (nopPutter) Put(context.Context, swarm.Chunk) error { return nil }

func TestSplitTreeShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level     redundancy.Level
		leaves    []int
		parities  []int
		rootData  int
		rootParit int
	}{
		{level: redundancy.MEDIUM, leaves: []int{75}, parities: []int{8}, rootData: 75, rootParit: 8},
		{level: redundancy.STRONG, leaves: []int{75}, parities: []int{17}, rootData: 75, rootParit: 17},
		{level: redundancy.INSANE, leaves: []int{75}, parities: []int{27}, rootData: 75, rootParit: 27},
		{level: redundancy.PARANOID, leaves: []int{39, 36}, parities: []int{89, 87}, rootData: 2, rootParit: 23},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("level_%d", tc.level), func(t *testing.T) {
			t.Parallel()
			_, tr := split(t, 307200, tc.level)

			leaves := tr.LeafNodes()
			if len(leaves) != len(tc.leaves) {
				t.Fatalf("leaf nodes: got %d, want %d", len(leaves), len(tc.leaves))
			}
			for i, n := range leaves {
				if len(n.Data) != tc.leaves[i] || len(n.Parities) != tc.parities[i] {
					t.Fatalf("leaf %d: got %d data + %d parities, want %d + %d", i, len(n.Data), len(n.Parities), tc.leaves[i], tc.parities[i])
				}
			}

			root := tr.Nodes[len(tr.Nodes)-1]
			if !root.Address.Equal(tr.Root) {
				t.Fatalf("last indexed node is %s, want root %s", root.Address, tr.Root)
			}
			if len(root.Data) != tc.rootData || len(root.Parities) != tc.rootParit {
				t.Fatalf("root: got %d data + %d parities, want %d + %d", len(root.Data), len(root.Parities), tc.rootData, tc.rootParit)
			}
			if len(tr.Replicas) != tc.level.GetReplicaCount() {
				t.Fatalf("replicas: got %d, want %d", len(tr.Replicas), tc.level.GetReplicaCount())
			}
		})
	}
}

func TestSplitRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := erasurecoding.Split(context.Background(), make([]byte, swarm.ChunkSize), redundancy.MEDIUM); err == nil {
		t.Fatal("expected error for single-chunk data")
	}
	if _, err := erasurecoding.Split(context.Background(), make([]byte, 2*swarm.ChunkSize), redundancy.NONE); err == nil {
		t.Fatal("expected error for level NONE")
	}
}

func TestDrops(t *testing.T) {
	t.Parallel()

	_, tr := split(t, 307200, redundancy.PARANOID)
	n := tr.LeafNodes()[1]

	t.Run("data shards first", func(t *testing.T) {
		t.Parallel()
		got, err := erasurecoding.Drops(n, len(n.Data)-1, random.PseudoGenerator(1))
		if err != nil {
			t.Fatal(err)
		}
		assertSubset(t, got, n.Data)
	})

	t.Run("parities after all data shards", func(t *testing.T) {
		t.Parallel()
		count := len(n.Data) + 5
		got, err := erasurecoding.Drops(n, count, random.PseudoGenerator(1))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != count {
			t.Fatalf("got %d drops, want %d", len(got), count)
		}
		assertSubset(t, got[:len(n.Data)], n.Data)
		assertSubset(t, got[len(n.Data):], n.Parities)
	})

	t.Run("deterministic for a seed", func(t *testing.T) {
		t.Parallel()
		a, _ := erasurecoding.Drops(n, 10, random.PseudoGenerator(7))
		b, _ := erasurecoding.Drops(n, 10, random.PseudoGenerator(7))
		for i := range a {
			if !a[i].Equal(b[i]) {
				t.Fatalf("drop %d differs: %s vs %s", i, a[i], b[i])
			}
		}
	})

	t.Run("too many", func(t *testing.T) {
		t.Parallel()
		if _, err := erasurecoding.Drops(n, len(n.Data)+len(n.Parities)+1, random.PseudoGenerator(1)); err == nil {
			t.Fatal("expected error")
		}
	})
}

func assertSubset(t *testing.T, got, set []swarm.Address) {
	t.Helper()
	seen := make(map[string]bool, len(got))
	for _, a := range got {
		if seen[a.ByteString()] {
			t.Fatalf("duplicate drop %s", a)
		}
		seen[a.ByteString()] = true
		found := false
		for _, s := range set {
			if s.Equal(a) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("drop %s not in expected set", a)
		}
	}
}

func TestRecoveryThreshold(t *testing.T) {
	t.Parallel()

	for _, level := range levels {
		for _, size := range []int{307200, 1 << 20} {
			t.Run(fmt.Sprintf("level_%d_size_%d", level, size), func(t *testing.T) {
				t.Parallel()
				data, tr := split(t, size, level)

				for i, n := range tr.LeafNodes() {
					m := len(n.Parities)

					dropped, err := erasurecoding.Drops(n, m, random.PseudoGenerator(int64(i)))
					if err != nil {
						t.Fatal(err)
					}
					got, err := join(t, tr.Store.Without(dropped), tr.Root, level)
					if err != nil {
						t.Fatalf("node %d: %d lost, want recovery, got error: %v", i, m, err)
					}
					if !bytes.Equal(got, data) {
						t.Fatalf("node %d: %d lost, recovered data differs", i, m)
					}

					if m+1 > len(n.Data)+len(n.Parities) {
						continue
					}
					dropped, err = erasurecoding.Drops(n, m+1, random.PseudoGenerator(int64(i)))
					if err != nil {
						t.Fatal(err)
					}
					got, err = join(t, tr.Store.Without(dropped), tr.Root, level)
					if err == nil && bytes.Equal(got, data) {
						t.Fatalf("node %d: %d lost, want failure, got full data", i, m+1)
					}
				}
			})
		}
	}
}

func TestRecoveryEveryLeafAtLimit(t *testing.T) {
	t.Parallel()

	for _, level := range levels {
		t.Run(fmt.Sprintf("level_%d", level), func(t *testing.T) {
			t.Parallel()
			data, tr := split(t, 1<<20, level)

			var dropped []swarm.Address
			for i, n := range tr.LeafNodes() {
				d, err := erasurecoding.Drops(n, len(n.Parities), random.PseudoGenerator(int64(i)))
				if err != nil {
					t.Fatal(err)
				}
				dropped = append(dropped, d...)
			}

			got, err := join(t, tr.Store.Without(dropped), tr.Root, level)
			if err != nil {
				t.Fatalf("want recovery, got error: %v", err)
			}
			if !bytes.Equal(got, data) {
				t.Fatal("recovered data differs")
			}
		})
	}
}

func BenchmarkSplit(b *testing.B) {
	data := randomData(b, 8<<20, 1)
	for _, simd := range []bool{false, true} {
		b.Run(fmt.Sprintf("simd_%t", simd), func(b *testing.B) {
			if simd {
				erasurecoding.EnableSIMD()
			}
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				if _, err := erasurecoding.Split(context.Background(), data, redundancy.MEDIUM); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

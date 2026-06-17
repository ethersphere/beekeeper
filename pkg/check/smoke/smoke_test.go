package smoke

import (
	"testing"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
)

func TestResolveRLevels(t *testing.T) {
	t.Run("empty defaults to single nil level", func(t *testing.T) {
		got := resolveRLevels(nil)
		if len(got) != 1 {
			t.Fatalf("expected 1 level, got %d", len(got))
		}
		if got[0] != nil {
			t.Fatalf("expected nil level, got %v", got[0])
		}
	})

	t.Run("empty non-nil slice defaults to single nil level", func(t *testing.T) {
		got := resolveRLevels([]*redundancy.Level{})
		if len(got) != 1 || got[0] != nil {
			t.Fatalf("expected single nil level, got %v", got)
		}
	})

	t.Run("returns configured levels unchanged", func(t *testing.T) {
		l := redundancy.Level(1)
		in := []*redundancy.Level{&l}
		got := resolveRLevels(in)
		if len(got) != 1 || got[0] != &l {
			t.Fatalf("expected configured levels returned unchanged, got %v", got)
		}
	})
}

func TestRedundancyLevelLabel(t *testing.T) {
	if got := redundancyLevelLabel(nil); got != "not_set" {
		t.Fatalf("nil: expected not_set, got %q", got)
	}
	l := redundancy.Level(2)
	if got := redundancyLevelLabel(&l); got != "2" {
		t.Fatalf("level 2: expected \"2\", got %q", got)
	}
}

func TestCountByteDiff(t *testing.T) {
	tests := []struct {
		name string
		a, b []byte
		want int
	}{
		{"equal", []byte{1, 2, 3}, []byte{1, 2, 3}, 0},
		{"all differ", []byte{1, 2, 3}, []byte{4, 5, 6}, 3},
		{"some differ", []byte{1, 2, 3}, []byte{1, 9, 3}, 1},
		{"shorter b compares min length", []byte{1, 2, 3}, []byte{1, 2}, 0},
		{"shorter a compares min length", []byte{1, 2}, []byte{1, 2, 3}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := countByteDiff(tc.a, tc.b); got != tc.want {
				t.Fatalf("countByteDiff(%v,%v)=%d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

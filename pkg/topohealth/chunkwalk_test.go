package topohealth

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

func TestSplitChunkAddresses_SmallFileSingleChunk(t *testing.T) {
	data := make([]byte, 1024) // < swarm.ChunkSize → fits in a single leaf chunk
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	root, chunks, err := SplitChunkAddresses(context.Background(), data, nil)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if root.Equal(swarm.ZeroAddress) {
		t.Fatal("root address is zero")
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Position != ChunkPositionRoot {
		t.Errorf("expected root position, got %s", chunks[0].Position)
	}
	if !chunks[0].Address.Equal(root) {
		t.Errorf("single chunk address %s != root %s", chunks[0].Address, root)
	}
}

func TestSplitChunkAddresses_MultiChunkTreeHasIntermediate(t *testing.T) {
	// Enough data to force at least one intermediate level: roughly
	// (BranchingFactor + 1) leaves = 129 chunks * 4096 = ~528KB.
	const size = 600 * 1024
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	root, chunks, err := SplitChunkAddresses(context.Background(), data, nil)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(chunks) < 130 {
		t.Fatalf("expected >130 chunks for %d bytes, got %d", size, len(chunks))
	}
	var roots, intermediates, leaves int
	for _, c := range chunks {
		switch c.Position {
		case ChunkPositionRoot:
			roots++
		case ChunkPositionIntermediate:
			intermediates++
		case ChunkPositionLeaf:
			leaves++
		}
	}
	if roots != 1 {
		t.Errorf("expected 1 root, got %d", roots)
	}
	if intermediates < 1 {
		t.Errorf("expected at least 1 intermediate chunk for %d bytes of data, got %d", size, intermediates)
	}
	if leaves < 130 {
		t.Errorf("expected at least 130 leaves, got %d", leaves)
	}
	// All chunks must be addressable.
	seen := make(map[string]bool, len(chunks))
	for _, c := range chunks {
		if c.Address.Equal(swarm.ZeroAddress) {
			t.Errorf("chunk has zero address")
		}
		if seen[c.Address.ByteString()] {
			t.Errorf("duplicate chunk address %s", c.Address)
		}
		seen[c.Address.ByteString()] = true
	}
	if !seen[root.ByteString()] {
		t.Errorf("root %s not in collected chunks", root)
	}
}

package gsoc

import (
	"context"
	"testing"
	"time"

	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

func TestMineResourceId(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		t.Fatal(err)
	}

	overlay, err := swarm.ParseHexAddress("aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	if err != nil {
		t.Fatal(err)
	}

	depth := 8
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nonce, socAddress, err := mineResourceId(ctx, overlay, privKey, depth)
	if err != nil {
		t.Fatalf("mineResourceId error: %v", err)
	}

	if len(nonce) != 32 {
		t.Fatalf("expected 32-byte nonce, got %d bytes", len(nonce))
	}

	prox := swarm.Proximity(socAddress.Bytes(), overlay.Bytes())
	if prox < uint8(depth) {
		t.Fatalf("expected proximity >= %d, got %d (soc: %s, overlay: %s)", depth, prox, socAddress, overlay)
	}
}

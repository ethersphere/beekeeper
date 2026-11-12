package pss

import (
	"testing"
	"time"
)

func TestNewDefaultOptions(t *testing.T) {
	t.Parallel()

	opts := NewDefaultOptions()
	if opts.AddressPrefix != 2 {
		t.Fatalf("expected AddressPrefix 2, got %d", opts.AddressPrefix)
	}
	if opts.PostageDepth != 22 {
		t.Fatalf("expected PostageDepth 22, got %d", opts.PostageDepth)
	}
	if opts.PostageTTL != 24*time.Hour {
		t.Fatalf("expected PostageTTL 24h, got %v", opts.PostageTTL)
	}
}

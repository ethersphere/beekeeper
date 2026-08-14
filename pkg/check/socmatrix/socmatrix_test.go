// Copyright 2026 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package socmatrix_test

import (
	"io"
	"testing"
	"time"

	"github.com/ethersphere/beekeeper/pkg/check/socmatrix"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

func TestNewDefaultOptions(t *testing.T) {
	opts := socmatrix.NewDefaultOptions()
	if opts.PostageTTL != 24*time.Hour {
		t.Fatalf("expected PostageTTL 24h, got %v", opts.PostageTTL)
	}
	if opts.PostageDepth != 16 {
		t.Fatalf("expected PostageDepth 16, got %d", opts.PostageDepth)
	}
	if opts.RequestTimeout != 10*time.Minute {
		t.Fatalf("expected RequestTimeout 10m, got %v", opts.RequestTimeout)
	}
	if opts.SyncWait != 45*time.Second {
		t.Fatalf("expected SyncWait 45s, got %v", opts.SyncWait)
	}
	if opts.Password != "beekeeper" {
		t.Fatalf("expected Password beekeeper, got %q", opts.Password)
	}
}

func TestNewCheck(t *testing.T) {
	chk := socmatrix.NewCheck(logging.New(io.Discard, 0))
	if chk == nil {
		t.Fatal("expected non-nil check")
	}
}

// Copyright 2026 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package socconvergence_test

import (
	"io"
	"testing"
	"time"

	"github.com/ethersphere/beekeeper/pkg/check/socconvergence"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

func TestNewDefaultOptions(t *testing.T) {
	opts := socconvergence.NewDefaultOptions()
	if opts.PostageTTL != 24*time.Hour {
		t.Fatalf("expected PostageTTL 24h, got %v", opts.PostageTTL)
	}
	if opts.PostageDepth != 16 {
		t.Fatalf("expected PostageDepth 16, got %d", opts.PostageDepth)
	}
	if opts.RequestTimeout != 5*time.Minute {
		t.Fatalf("expected RequestTimeout 5m, got %v", opts.RequestTimeout)
	}
}

func TestNewCheck(t *testing.T) {
	chk := socconvergence.NewCheck(logging.New(io.Discard, 0))
	if chk == nil {
		t.Fatal("expected non-nil check")
	}
}

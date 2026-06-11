package config_test

import (
	"io"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

// TestBeeConfigInheritance verifies that _inherit still merges per flag (not
// all-or-nothing) now that the flags live in an embedded orchestration.Config:
// the child inherits the parent's unset flags while keeping its own overrides.
func TestBeeConfigInheritance(t *testing.T) {
	t.Parallel()

	const in = `
bee-configs:
  default:
    api-addr: ":1633"
    full-node: true
    verbosity: 5
  child:
    _inherit: default
    full-node: false
`
	cfg, err := config.Read(logging.New(io.Discard, 0), []config.YamlFile{{Name: "test.yaml", Content: []byte(in)}})
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	child, ok := cfg.BeeConfigs["child"]
	if !ok {
		t.Fatal("child bee-config not found")
	}
	c := child.Export()

	// inherited from the parent (unset in the child)
	if c.APIAddr == nil || *c.APIAddr != ":1633" {
		t.Errorf("APIAddr = %v, want inherited :1633", c.APIAddr)
	}
	if c.Verbosity == nil || *c.Verbosity != "5" {
		t.Errorf("Verbosity = %v, want inherited \"5\"", c.Verbosity)
	}
	// the child's own explicit value wins over the parent, per flag
	if c.FullNode == nil || *c.FullNode {
		t.Errorf("FullNode = %v, want child override false", c.FullNode)
	}
}

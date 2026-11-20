package config_test

import (
	"io"
	"os"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"gopkg.in/yaml.v3"
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

// TestLocalDNSLegacyNodeGroup verifies that the local-dns cluster runs one node
// group in bee's legacy regime next to the node-mode groups: the legacy
// profile overrides the inherited node-mode with an empty value, so bee infers
// the mode from full-node, while its siblings keep the explicit mode.
func TestLocalDNSLegacyNodeGroup(t *testing.T) {
	t.Parallel()

	in, err := os.ReadFile("../../config/local.yaml")
	if err != nil {
		t.Fatalf("read local.yaml: %v", err)
	}
	cfg, err := config.Read(logging.New(io.Discard, 0), []config.YamlFile{{Name: "local.yaml", Content: in}})
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	cluster, ok := cfg.Clusters["local-dns"]
	if !ok {
		t.Fatal("local-dns cluster not found")
	}
	ng, ok := cluster.GetNodeGroups()["bee-legacy"]
	if !ok {
		t.Fatal("bee-legacy node group not found in local-dns")
	}

	legacy, ok := cfg.BeeConfigs[ng.BeeConfig]
	if !ok {
		t.Fatalf("bee-config %q not found", ng.BeeConfig)
	}
	l := legacy.Export()
	if l.NodeMode == nil {
		t.Error("NodeMode = nil, want explicit empty override")
	} else if *l.NodeMode != "" {
		t.Errorf("NodeMode = %q, want explicit empty override", *l.NodeMode)
	}
	if l.FullNode == nil || !*l.FullNode {
		t.Errorf("FullNode = %v, want true", l.FullNode)
	}
	if !l.IsFullNode() {
		t.Error("IsFullNode() = false, want true from the full-node fallback")
	}

	out, err := yaml.Marshal(l)
	if err != nil {
		t.Fatalf("marshal bee config: %v", err)
	}
	rendered := map[string]any{}
	if err := yaml.Unmarshal(out, &rendered); err != nil {
		t.Fatalf("unmarshal rendered config: %v", err)
	}
	if got, ok := rendered["node-mode"]; !ok || got != "" {
		t.Errorf("rendered node-mode = %v (present %t), want empty string so bee selects the legacy regime", got, ok)
	}
	if got := rendered["full-node"]; got != true {
		t.Errorf("rendered full-node = %v, want true", got)
	}

	dns := cfg.BeeConfigs["bee-local-dns"]
	modern := dns.Export()
	if !modern.IsFullNode() || modern.NodeMode == nil || *modern.NodeMode != "full" {
		t.Error("bee-local-dns should keep the explicit node-mode full")
	}
}

package config_test

import (
	"testing"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"gopkg.in/yaml.v3"
)

func TestBeeConfigExport(t *testing.T) {
	t.Parallel()

	// Export returns the embedded orchestration.Config verbatim: set fields
	// (including explicit zero values) are preserved, unset ones stay nil.
	in := config.BeeConfig{
		Config: orchestration.Config{
			APIAddr:    new(":1633"),
			FullNode:   new(false),            // explicit zero value must survive
			WarmupTime: new(time.Duration(0)), // explicit 0s must survive
		},
	}

	got := in.Export()

	if got.APIAddr == nil || *got.APIAddr != ":1633" {
		t.Errorf("APIAddr = %v, want :1633", got.APIAddr)
	}
	if got.FullNode == nil || *got.FullNode {
		t.Errorf("FullNode = %v, want explicit false preserved", got.FullNode)
	}
	if got.WarmupTime == nil || *got.WarmupTime != 0 {
		t.Errorf("WarmupTime = %v, want 0s preserved", got.WarmupTime)
	}
	if got.P2PAddr != nil {
		t.Errorf("P2PAddr = %v, want nil (unset)", *got.P2PAddr)
	}
}

// TestBeeConfigRender exercises the full pipeline a node config goes through:
// Beekeeper YAML -> config.BeeConfig -> Export -> yaml.Marshal -> .bee.yaml.
// The config keys are the Bee flag names, and _inherit is read but must never
// reach the rendered file.
func TestBeeConfigRender(t *testing.T) {
	t.Parallel()

	const in = `
_inherit: default
api-addr: ":1633"
p2p-addr: ":1634"
full-node: false
warmup-time: 0s
bootnode: ["/dnsaddr/localhost"]
payment-threshold: 13500000
tracing-enable: false
`
	var bc config.BeeConfig
	if err := yaml.Unmarshal([]byte(in), &bc); err != nil {
		t.Fatalf("unmarshal beekeeper config: %v", err)
	}
	if bc.GetParentName() != "default" {
		t.Fatalf("GetParentName() = %q, want default (read from _inherit)", bc.GetParentName())
	}

	out, err := yaml.Marshal(bc.Export())
	if err != nil {
		t.Fatalf("marshal bee config: %v", err)
	}

	rendered := map[string]any{}
	if err := yaml.Unmarshal(out, &rendered); err != nil {
		t.Fatalf("unmarshal rendered config: %v", err)
	}

	// Set keys are present, with explicit zero values preserved. Note
	// payment-threshold: a numeric YAML scalar is read into bee's string-typed
	// flag and rendered back as a string.
	wantPresent := map[string]any{
		"api-addr":          ":1633",
		"p2p-addr":          ":1634",
		"full-node":         false,
		"warmup-time":       "0s",
		"payment-threshold": "13500000",
		"tracing-enable":    false,
	}
	for k, want := range wantPresent {
		got, ok := rendered[k]
		if !ok {
			t.Errorf("rendered .bee.yaml is missing key %q", k)
			continue
		}
		if got != want {
			t.Errorf("rendered[%q] = %v (%T), want %v (%T)", k, got, got, want, want)
		}
	}

	// bootnode is a list flag in bee and must render as a YAML list.
	if v, ok := rendered["bootnode"]; !ok {
		t.Error("rendered .bee.yaml is missing key \"bootnode\"")
	} else if l, isList := v.([]any); !isList || len(l) != 1 || l[0] != "/dnsaddr/localhost" {
		t.Errorf(`bootnode = %v, want ["/dnsaddr/localhost"]`, v)
	}

	// Unset flags are omitted so Bee applies its own defaults, and _inherit (a
	// config-loading concern) never leaks into the rendered file.
	for _, k := range []string{"verbosity", "mainnet", "swap-enable", "password", "_inherit"} {
		if _, ok := rendered[k]; ok {
			t.Errorf("rendered .bee.yaml should omit %q, but it is present", k)
		}
	}
}

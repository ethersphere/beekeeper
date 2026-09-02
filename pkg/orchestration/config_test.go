package orchestration_test

import (
	"testing"
	"time"

	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"gopkg.in/yaml.v3"
)

// TestConfigMarshal documents the contract the rendered .bee.yaml relies on:
// a nil field is omitted (Bee uses its own default), a non-nil field is emitted
// even when it points to a zero value, and the Bee flag names are used.
func TestConfigMarshal(t *testing.T) {
	t.Parallel()

	cfg := orchestration.Config{
		FullNode:   new(false),            // zero value, explicitly set -> emitted
		WarmupTime: new(time.Duration(0)), // 0s, explicitly set -> emitted as "0s"
		Bootnodes:  &[]string{},           // empty list, explicitly set -> emitted (overrides bee's default bootnodes)
		// all other fields nil -> omitted
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	m := map[string]any{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(m) != 3 {
		t.Fatalf("expected exactly the 3 set fields to be rendered, got %d: %v", len(m), m)
	}
	if v, ok := m["full-node"]; !ok || v != false {
		t.Errorf("full-node = %v (present=%v), want false", v, ok)
	}
	if v, ok := m["warmup-time"]; !ok || v != "0s" {
		t.Errorf(`warmup-time = %v (present=%v), want "0s"`, v, ok)
	}
	if v, ok := m["bootnode"]; !ok {
		t.Error("bootnode missing, want explicit empty list to be emitted")
	} else if l, isList := v.([]any); !isList || len(l) != 0 {
		t.Errorf("bootnode = %v, want empty list", v)
	}
}

func TestDeref(t *testing.T) {
	t.Parallel()

	if got := orchestration.Deref[string](nil); got != "" {
		t.Errorf("Deref(nil) = %q, want empty string", got)
	}
	if got := orchestration.Deref(new(true)); !got {
		t.Errorf("Deref(new(true)) = %v, want true", got)
	}
	if got := orchestration.Deref(new(false)); got {
		t.Errorf("Deref(new(false)) = %v, want false", got)
	}
}

func TestNodeModeHelpers(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name           string
		cfg            orchestration.Config
		wantFull       bool
		wantLight      bool
		wantUltraLight bool
	}{
		{
			name:           "explicit node-mode full",
			cfg:            orchestration.Config{NodeMode: strPtr("full")},
			wantFull:       true,
			wantLight:      false,
			wantUltraLight: false,
		},
		{
			name:           "explicit node-mode light",
			cfg:            orchestration.Config{NodeMode: strPtr("light")},
			wantFull:       false,
			wantLight:      true,
			wantUltraLight: false,
		},
		{
			name:           "explicit node-mode ultra-light",
			cfg:            orchestration.Config{NodeMode: strPtr("ultra-light")},
			wantFull:       false,
			wantLight:      false,
			wantUltraLight: true,
		},
		{
			name:           "legacy full-node true",
			cfg:            orchestration.Config{FullNode: boolPtr(true)},
			wantFull:       true,
			wantLight:      false,
			wantUltraLight: false,
		},
		{
			name: "legacy full-node false with rpc (light)",
			cfg: orchestration.Config{
				FullNode:              boolPtr(false),
				BlockchainRPCEndpoint: strPtr("http://localhost:8545"),
			},
			wantFull:       false,
			wantLight:      true,
			wantUltraLight: false,
		},
		{
			name:           "legacy full-node false without rpc (ultra-light)",
			cfg:            orchestration.Config{FullNode: boolPtr(false)},
			wantFull:       false,
			wantLight:      false,
			wantUltraLight: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsFullNode(); got != tt.wantFull {
				t.Errorf("IsFullNode() = %v, want %v", got, tt.wantFull)
			}
			if got := tt.cfg.IsLightNode(); got != tt.wantLight {
				t.Errorf("IsLightNode() = %v, want %v", got, tt.wantLight)
			}
			if got := tt.cfg.IsUltraLightNode(); got != tt.wantUltraLight {
				t.Errorf("IsUltraLightNode() = %v, want %v", got, tt.wantUltraLight)
			}
		})
	}
}

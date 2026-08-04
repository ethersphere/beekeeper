package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

func TestSOCPartitionClusterConfig(t *testing.T) {
	t.Parallel()

	dir := filepath.Join("..", "..", "config")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var files []config.YamlFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, config.YamlFile{Name: e.Name(), Content: content})
	}

	cfg, err := config.Read(logging.New(os.Stderr, 0), files)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	cluster, ok := cfg.Clusters["local-soc-partition"]
	if !ok {
		t.Fatal("local-soc-partition cluster missing")
	}
	groups := cluster.GetNodeGroups()
	if _, ok := groups["bootnode"]; !ok {
		t.Fatal("bootnode group missing")
	}
	if g, ok := groups["group-a"]; !ok || g.Count != 2 {
		t.Fatalf("group-a: %+v", g)
	}
	if g, ok := groups["group-b"]; !ok || len(g.Nodes) != 2 {
		t.Fatalf("group-b: %+v", g)
	}

	a, ok := cfg.BeeConfigs["bee-soc-partition-group-a"]
	if !ok || a.TargetNeighborhood == nil || *a.TargetNeighborhood != "11111100" {
		t.Fatalf("group-a bee config: %+v", a)
	}
	b, ok := cfg.BeeConfigs["bee-soc-partition-group-b"]
	if !ok || b.Bootnodes == nil || len(*b.Bootnodes) == 0 {
		t.Fatalf("group-b must have non-empty bootnodes: %+v", b)
	}
	if b.TargetNeighborhood == nil || *b.TargetNeighborhood != "11111100" {
		t.Fatalf("group-b target neighborhood: %+v", b)
	}

	exported := b.Export()
	if exported.TargetNeighborhood == nil || *exported.TargetNeighborhood != "11111100" {
		t.Fatalf("export target neighborhood: %v", exported.TargetNeighborhood)
	}
	if exported.Bootnodes == nil || len(*exported.Bootnodes) == 0 {
		t.Fatal("export bootnodes empty")
	}
}

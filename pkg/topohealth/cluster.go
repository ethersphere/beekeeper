package topohealth

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

type ClusterCounts struct {
	FullNodes  int `json:"fullNodes"`
	LightNodes int `json:"lightNodes"`
	Bootnodes  int `json:"bootnodes"`
}

// ClusterShape inspects node configs only; no API calls.
func ClusterShape(c orchestration.Cluster) ClusterCounts {
	var cc ClusterCounts
	for _, n := range c.Nodes() {
		cfg := n.Config()
		switch {
		case cfg.BootnodeMode:
			cc.Bootnodes++
		case cfg.FullNode:
			cc.FullNodes++
		default:
			cc.LightNodes++
		}
	}
	return cc
}

type StorerResult struct {
	Verdict  Verdict `json:"verdict"`
	HasChunk bool    `json:"hasChunk"`
	// HasError is set if HEAD /chunks/{addr} or the topology probe failed.
	// A probe error leaves Verdict.Status == StatusUnknown (not Unhealthy);
	// callers can distinguish "probe failed" from "node is genuinely unhealthy".
	HasError string `json:"hasError,omitempty"`
}

// IntendedStorers returns probe results for the top-n full nodes closest to
// chunkAddr, including a local HEAD /chunks/{addr} check on each. Probes run
// concurrently.
func IntendedStorers(ctx context.Context, c orchestration.Cluster, chunkAddr swarm.Address, n int, t Thresholds) ([]StorerResult, error) {
	clients, err := c.FullNodeClientsByDistance(ctx, chunkAddr)
	if err != nil {
		return nil, fmt.Errorf("rank full nodes by distance: %w", err)
	}
	if n > len(clients) {
		n = len(clients)
	}
	results := make([]StorerResult, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int, cl *bee.Client) {
			defer wg.Done()
			results[idx] = probeStorer(ctx, cl, chunkAddr, t)
		}(i, clients[i])
	}
	wg.Wait()
	return results, nil
}

func probeStorer(ctx context.Context, cl *bee.Client, chunkAddr swarm.Address, t Thresholds) StorerResult {
	v, err := Probe(ctx, cl, t)
	if err != nil {
		return StorerResult{
			Verdict:  Verdict{Node: cl.Name(), Status: StatusUnknown},
			HasError: "probe: " + err.Error(),
		}
	}
	has, herr := cl.LocalHasChunk(ctx, chunkAddr)
	r := StorerResult{Verdict: v, HasChunk: has}
	if herr != nil {
		r.HasError = "has_chunk: " + herr.Error()
	}
	return r
}

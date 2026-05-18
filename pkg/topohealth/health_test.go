package topohealth

import (
	"testing"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
)

func healthyStatus() *api.StatusResponse {
	return &api.StatusResponse{
		BeeMode:        api.BeeModeFull,
		IsReachable:    true,
		IsWarmingUp:    false,
		ReserveSize:    1000,
		StorageRadius:  3,
		CommittedDepth: 3,
	}
}

func healthyTopology() bee.Topology {
	bins := bee.BinMap{}
	// bins below depth 3 each have 4 connected (saturated), 0 disconnected.
	for i := range 3 {
		bins[binKey(i)] = bee.Bin{Connected: 4, Population: 4}
	}
	// neighborhood (bin >= depth)
	bins[binKey(3)] = bee.Bin{Connected: 3, Population: 3}
	return bee.Topology{
		Depth:        3,
		Connected:    15,
		Population:   15,
		Reachability: "Public",
		Bins:         bins,
	}
}

func TestEvaluate_Healthy(t *testing.T) {
	v := Evaluate("n1", healthyStatus(), healthyTopology(), DefaultThresholds())
	if v.Status != StatusHealthy {
		t.Fatalf("expected HEALTHY, got %s, reasons=%v", v.Status, v.FailureReasons)
	}
}

func TestEvaluate_EmptyBinBelowDepth_Unhealthy(t *testing.T) {
	top := healthyTopology()
	top.Bins[binKey(1)] = bee.Bin{Connected: 0, Population: 0}
	v := Evaluate("n1", healthyStatus(), top, DefaultThresholds())
	if v.Status != StatusUnhealthy {
		t.Fatalf("expected UNHEALTHY for missing bin below depth, got %s", v.Status)
	}
	if v.Raw.EmptyBinsBelowDepth != 1 {
		t.Errorf("EmptyBinsBelowDepth = %d, want 1", v.Raw.EmptyBinsBelowDepth)
	}
}

func TestEvaluate_LowSaturation_Degraded(t *testing.T) {
	top := healthyTopology()
	top.Bins[binKey(0)] = bee.Bin{Connected: 2, Population: 2}
	v := Evaluate("n1", healthyStatus(), top, DefaultThresholds())
	if v.Status != StatusDegraded {
		t.Fatalf("expected DEGRADED for under-saturated bin, got %s, reasons=%v", v.Status, v.FailureReasons)
	}
}

func TestEvaluate_WarmingUp_Unhealthy(t *testing.T) {
	st := healthyStatus()
	st.IsWarmingUp = true
	v := Evaluate("n1", st, healthyTopology(), DefaultThresholds())
	if v.Status != StatusUnhealthy {
		t.Fatalf("expected UNHEALTHY while warming up, got %s", v.Status)
	}
}

func TestEvaluate_NotPublic_RequiresPublic(t *testing.T) {
	top := healthyTopology()
	top.Reachability = "Private"
	v := Evaluate("n1", healthyStatus(), top, DefaultThresholds())
	if v.Status != StatusUnhealthy {
		t.Fatalf("expected UNHEALTHY for non-Public reachability when RequirePublicReachable=true, got %s", v.Status)
	}
	// And HEALTHY when RequirePublicReachable=false.
	thr := DefaultThresholds()
	thr.RequirePublicReachable = false
	v = Evaluate("n1", healthyStatus(), top, thr)
	if v.Status != StatusHealthy {
		t.Fatalf("expected HEALTHY with RequirePublicReachable=false, got %s, reasons=%v", v.Status, v.FailureReasons)
	}
}

func TestEvaluate_BinsViewIncludesEmpty(t *testing.T) {
	top := healthyTopology()
	delete(top.Bins, binKey(2)) // simulate bin_2 entirely missing from response
	v := Evaluate("n1", healthyStatus(), top, DefaultThresholds())
	var found bool
	for _, b := range v.Bins {
		if b.Index == 2 {
			found = true
			if !b.Empty || b.Connected != 0 {
				t.Errorf("bin_2 should be empty with 0 connected, got %+v", b)
			}
		}
	}
	if !found {
		t.Fatalf("expected bin_2 in the view even though missing from response")
	}
}

func TestEvaluate_LightNodeNotPenalizedForReserve(t *testing.T) {
	st := healthyStatus()
	st.BeeMode = api.BeeModeLight
	st.ReserveSize = 0
	st.StorageRadius = 0
	st.CommittedDepth = 0
	v := Evaluate("n1", st, healthyTopology(), DefaultThresholds())
	if v.Status != StatusHealthy {
		t.Fatalf("light node with empty reserve should be HEALTHY, got %s, reasons=%v", v.Status, v.FailureReasons)
	}
}

// Package topohealth implements a topology health probe over a Bee cluster.
// It calls /status and /topology on individual nodes, computes a set of
// signals, and produces a Verdict (healthy/degraded/unhealthy) used by the
// smoke tests to diagnose retrieval failures.
package topohealth

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
)

// Status is the rolled-up verdict for a node. The zero value is StatusUnknown
// so a default-initialized Verdict is never silently read as healthy/unhealthy.
type Status int

const (
	StatusUnknown Status = iota
	StatusUnhealthy
	StatusDegraded
	StatusHealthy
)

func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "HEALTHY"
	case StatusDegraded:
		return "DEGRADED"
	case StatusUnhealthy:
		return "UNHEALTHY"
	default:
		return "UNKNOWN"
	}
}

func (s Status) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `%q`, s.String()), nil
}

func (s *Status) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch strings.ToUpper(v) {
	case "HEALTHY":
		*s = StatusHealthy
	case "DEGRADED":
		*s = StatusDegraded
	case "UNHEALTHY":
		*s = StatusUnhealthy
	default:
		*s = StatusUnknown
	}
	return nil
}

type Signals struct {
	DepthOK              bool    `json:"depthOK"`
	BelowDepthFilled     bool    `json:"belowDepthFilled"`
	BelowDepthSaturation float64 `json:"belowDepthSaturation"`
	NeighborhoodSizeOK   bool    `json:"neighborhoodSizeOK"`
	Reachable            bool    `json:"reachable"`
	WarmedUp             bool    `json:"warmedUp"`
	DialRatioOK          bool    `json:"dialRatioOK"`
	// ReserveAligned/ReserveNonEmpty are only meaningful for full nodes.
	ReserveAligned  bool `json:"reserveAligned"`
	ReserveNonEmpty bool `json:"reserveNonEmpty"`
}

// BinView is a per-bin snapshot. Unlike bee.BinMap.String it includes empty
// bins, which is the signal we need for "is there a gap below depth?".
type BinView struct {
	Index        int  `json:"index"`
	Connected    int  `json:"connected"`
	Disconnected int  `json:"disconnected"`
	Population   int  `json:"population"`
	Empty        bool `json:"empty"`
}

type Raw struct {
	Depth                   int     `json:"depth"`
	NnLowWatermark          int     `json:"nnLowWatermark"`
	Connected               int     `json:"connected"`
	Population              int     `json:"population"`
	NeighborhoodSize        int     `json:"neighborhoodSize"`
	EmptyBinsBelowDepth     int     `json:"emptyBinsBelowDepth"`
	DialRatio               float64 `json:"dialRatio"`
	BeeMode                 string  `json:"beeMode"`
	IsReachable             bool    `json:"isReachable"`
	IsWarmingUp             bool    `json:"isWarmingUp"`
	Reachability            string  `json:"reachability"`
	NetworkAvailability     string  `json:"networkAvailability"`
	ReserveSize             uint64  `json:"reserveSize"`
	ReserveSizeWithinRadius uint64  `json:"reserveSizeWithinRadius"`
	StorageRadius           uint8   `json:"storageRadius"`
	CommittedDepth          uint8   `json:"committedDepth"`
	PullsyncRate            float64 `json:"pullsyncRate"`
}

type Verdict struct {
	Node           string        `json:"node"`
	Overlay        swarm.Address `json:"overlay"`
	Status         Status        `json:"status"`
	Signals        Signals       `json:"signals"`
	Raw            Raw           `json:"raw"`
	Bins           []BinView     `json:"bins"`
	FailureReasons []string      `json:"failureReasons,omitempty"`
}

// Probe runs the single-node probe: /status and /topology in parallel.
func Probe(ctx context.Context, c *bee.Client, t Thresholds) (Verdict, error) {
	var (
		wg     sync.WaitGroup
		st     *api.StatusResponse
		top    bee.Topology
		stErr  error
		topErr error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		st, stErr = c.Status(ctx)
	}()
	go func() {
		defer wg.Done()
		top, topErr = c.Topology(ctx)
	}()
	wg.Wait()
	if stErr != nil {
		return Verdict{}, fmt.Errorf("status: %w", stErr)
	}
	if topErr != nil {
		return Verdict{}, fmt.Errorf("topology: %w", topErr)
	}
	return Evaluate(c.Name(), st, top, t), nil
}

// Evaluate is the pure rule-table portion of Probe, exported so the rule logic
// can be tested without a live bee node.
func Evaluate(nodeName string, st *api.StatusResponse, top bee.Topology, t Thresholds) Verdict {
	bins, walk := walkBins(top, t)
	signals := computeSignals(st, top, walk, t)
	status, reasons := decideStatus(st, top, signals, walk)

	return Verdict{
		Node:    nodeName,
		Overlay: top.Overlay,
		Status:  status,
		Signals: signals,
		Bins:    bins,
		Raw: Raw{
			Depth:                   top.Depth,
			NnLowWatermark:          top.NnLowWatermark,
			Connected:               top.Connected,
			Population:              top.Population,
			NeighborhoodSize:        int(st.NeighborhoodSize),
			EmptyBinsBelowDepth:     walk.emptyBelow,
			DialRatio:               walk.dialRatio,
			BeeMode:                 st.BeeMode,
			IsReachable:             st.IsReachable,
			IsWarmingUp:             st.IsWarmingUp,
			Reachability:            top.Reachability,
			NetworkAvailability:     top.NetworkAvailability,
			ReserveSize:             st.ReserveSize,
			ReserveSizeWithinRadius: st.ReserveSizeWithinRadius,
			StorageRadius:           st.StorageRadius,
			CommittedDepth:          st.CommittedDepth,
			PullsyncRate:            st.PullsyncRate,
		},
		FailureReasons: reasons,
	}
}

type binWalkResult struct {
	emptyBelow        int
	minBelowSat       float64
	belowFilled       bool
	neighborhoodPeers int
	dialRatio         float64
}

// walkBins iterates bins from 0..maxBin once and produces both the per-bin
// view and every aggregate the verdict rules need, including saturation
// (which depends on the threshold).
func walkBins(top bee.Topology, t Thresholds) ([]BinView, binWalkResult) {
	maxBin := top.Depth
	for k := range top.Bins {
		idx, ok := parseBinIndex(k)
		if !ok {
			continue
		}
		if idx > maxBin {
			maxBin = idx
		}
	}

	bins := make([]BinView, 0, maxBin+1)
	r := binWalkResult{
		minBelowSat: -1,
		belowFilled: true,
	}
	totalConnected, totalDisconnected := 0, 0

	for i := 0; i <= maxBin; i++ {
		b := top.Bins[binKey(i)]
		disc := len(b.DisconnectedPeers)
		bins = append(bins, BinView{
			Index:        i,
			Connected:    b.Connected,
			Disconnected: disc,
			Population:   b.Population,
			Empty:        b.Connected == 0 && b.Population == 0,
		})
		totalConnected += b.Connected
		totalDisconnected += disc

		if i < top.Depth {
			if b.Connected == 0 {
				r.belowFilled = false
				r.emptyBelow++
			}
			sat := float64(b.Connected) / float64(t.SaturationPeers)
			if r.minBelowSat < 0 || sat < r.minBelowSat {
				r.minBelowSat = sat
			}
		} else {
			r.neighborhoodPeers += b.Connected
		}
	}
	if r.minBelowSat < 0 {
		r.minBelowSat = 1.0
	}

	r.dialRatio = 1.0
	if totalConnected+totalDisconnected > 0 {
		r.dialRatio = float64(totalConnected) / float64(totalConnected+totalDisconnected)
	}
	return bins, r
}

func binKey(i int) string {
	return "bin_" + strconv.Itoa(i)
}

func parseBinIndex(k string) (int, bool) {
	rest, ok := strings.CutPrefix(k, "bin_")
	if !ok {
		return 0, false
	}
	idx, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}
	return idx, true
}

func computeSignals(st *api.StatusResponse, top bee.Topology, w binWalkResult, t Thresholds) Signals {
	reachable := st.IsReachable
	if t.RequirePublicReachable {
		reachable = reachable && strings.EqualFold(top.Reachability, "Public")
	}

	sig := Signals{
		DepthOK:              top.Depth > 0,
		BelowDepthFilled:     w.belowFilled,
		BelowDepthSaturation: w.minBelowSat,
		NeighborhoodSizeOK:   w.neighborhoodPeers >= max(t.MinNeighborhoodSize, top.NnLowWatermark),
		Reachable:            reachable,
		WarmedUp:             !st.IsWarmingUp,
		DialRatioOK:          w.dialRatio >= t.MinDialRatio,
	}

	if isFullNode(st.BeeMode) {
		sig.ReserveAligned = sig.WarmedUp && st.StorageRadius == st.CommittedDepth
		sig.ReserveNonEmpty = st.ReserveSize > 0
	} else {
		// Light/ultra-light: marked satisfied so the verdict isn't tripped by
		// fields that don't apply to that mode.
		sig.ReserveAligned = true
		sig.ReserveNonEmpty = true
	}
	return sig
}

func isFullNode(beeMode string) bool {
	return strings.EqualFold(beeMode, api.BeeModeFull)
}

// decideStatus applies the rule table:
//
//	UNHEALTHY: any of DepthOK / BelowDepthFilled / NeighborhoodSizeOK /
//	           Reachable / WarmedUp is false.
//	DEGRADED:  BelowDepthSaturation < 1.0 OR DialRatioOK false OR
//	           (full node && !ReserveAligned).
//	HEALTHY:   otherwise.
func decideStatus(st *api.StatusResponse, top bee.Topology, sig Signals, w binWalkResult) (Status, []string) {
	var hard, soft []string
	if !sig.DepthOK {
		hard = append(hard, "depth_not_ok")
	}
	if !sig.BelowDepthFilled {
		hard = append(hard, fmt.Sprintf("empty_bins_below_depth=%d", w.emptyBelow))
	}
	if !sig.NeighborhoodSizeOK {
		hard = append(hard, fmt.Sprintf("neighborhood_size=%d<min", w.neighborhoodPeers))
	}
	if !sig.Reachable {
		hard = append(hard, fmt.Sprintf("unreachable(reachability=%q)", top.Reachability))
	}
	if !sig.WarmedUp {
		hard = append(hard, "warming_up")
	}
	if sig.BelowDepthSaturation < 1.0 {
		soft = append(soft, fmt.Sprintf("below_depth_sat=%.2f", sig.BelowDepthSaturation))
	}
	if !sig.DialRatioOK {
		soft = append(soft, fmt.Sprintf("dial_ratio=%.2f<min", w.dialRatio))
	}
	if isFullNode(st.BeeMode) && !sig.ReserveAligned {
		soft = append(soft, fmt.Sprintf("reserve_misaligned(storage=%d,committed=%d)", st.StorageRadius, st.CommittedDepth))
	}

	switch {
	case len(hard) > 0:
		reasons := append(hard, soft...)
		sort.Strings(reasons)
		return StatusUnhealthy, reasons
	case len(soft) > 0:
		sort.Strings(soft)
		return StatusDegraded, soft
	default:
		return StatusHealthy, nil
	}
}

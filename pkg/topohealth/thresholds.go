package topohealth

// Thresholds tunes the rules used to derive a per-node Verdict.
// Defaults match a production Swarm cluster; small testnets may need lower
// SaturationPeers.
type Thresholds struct {
	// SaturationPeers is the minimum number of connected peers per bin below
	// the node's depth for that bin to be considered saturated.
	SaturationPeers int
	// MinNeighborhoodSize is the minimum number of peers within the node's
	// neighborhood (bins >= depth) for the node to be considered usable.
	MinNeighborhoodSize int
	// MinDialRatio is the minimum connected / (connected + disconnected) ratio
	// across all bins under which the node is flagged as having dial issues.
	MinDialRatio float64
	// RequirePublicReachable, when true, requires reachability == "Public"
	// (instead of just isReachable == true).
	RequirePublicReachable bool
}

// DefaultThresholds returns the production defaults.
func DefaultThresholds() Thresholds {
	return Thresholds{
		SaturationPeers:        4,
		MinNeighborhoodSize:    2,
		MinDialRatio:           0.8,
		RequirePublicReachable: true,
	}
}

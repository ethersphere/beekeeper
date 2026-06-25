package api

import (
	"context"
	"net/http"
)

type RedistributionService service

// RedistributionState is the subset of the node's /redistributionstate response
// relevant to radius / pull-sync validation: whether the node is frozen, fully
// synced, healthy, and how far the redistribution round has progressed. These are
// the signals that distinguish a recovering node from a halted one after a radius
// change (a node that cannot reach fullySynced or that freezes skips rounds).
type RedistributionState struct {
	IsFrozen                  bool    `json:"isFrozen"`
	IsFullySynced             bool    `json:"isFullySynced"`
	IsHealthy                 bool    `json:"isHealthy"`
	HasSufficientFunds        bool    `json:"hasSufficientFunds"`
	Phase                     string  `json:"phase"`
	Round                     uint64  `json:"round"`
	LastWonRound              uint64  `json:"lastWonRound"`
	LastPlayedRound           uint64  `json:"lastPlayedRound"`
	LastFrozenRound           uint64  `json:"lastFrozenRound"`
	LastSelectedRound         uint64  `json:"lastSelectedRound"`
	LastSampleDurationSeconds float64 `json:"lastSampleDurationSeconds"`
	Block                     uint64  `json:"block"`
}

// RedistributionState returns the node's redistribution-game state. This endpoint
// is full-mode only; it errors on nodes without the redistribution agent enabled.
func (s *RedistributionService) RedistributionState(ctx context.Context) (resp *RedistributionState, err error) {
	err = s.client.requestJSON(ctx, http.MethodGet, "/redistributionstate", nil, &resp)
	return resp, err
}

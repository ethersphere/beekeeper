package api

import (
	"context"
	"net/http"
)

type StatusService service

type StatusResponse struct {
	Overlay                 string  `json:"overlay"`
	Proximity               uint    `json:"proximity"`
	BeeMode                 string  `json:"beeMode"`
	ReserveSize             uint64  `json:"reserveSize"`
	ReserveSizeWithinRadius uint64  `json:"reserveSizeWithinRadius"`
	PullsyncRate            float64 `json:"pullsyncRate"`
	StorageRadius           uint8   `json:"storageRadius"`
	ConnectedPeers          uint64  `json:"connectedPeers"`
	NeighborhoodSize        uint64  `json:"neighborhoodSize"`
	RequestFailed           bool    `json:"requestFailed,omitempty"`
	BatchCommitment         uint64  `json:"batchCommitment"`
	IsReachable             bool    `json:"isReachable"`
	LastSyncedBlock         uint64  `json:"lastSyncedBlock"`
	CommittedDepth          uint8   `json:"committedDepth"`
}

// Ping pings given node
func (s *StatusService) Status(ctx context.Context) (resp *StatusResponse, err error) {
	err = s.client.requestJSON(ctx, http.MethodGet, "/status", nil, &resp)
	return resp, err
}

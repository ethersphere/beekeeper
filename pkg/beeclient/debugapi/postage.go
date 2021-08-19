package debugapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethersphere/beekeeper/pkg/bigint"
)

// PostageService represents Bee's Postage service
type PostageService service

type postageResponse struct {
	BatchID string `json:"batchID"`
}

type PostageStampResponse struct {
	BatchID       string         `json:"batchID"`
	Utilization   uint32         `json:"utilization"`
	Usable        bool           `json:"usable"`
	Label         string         `json:"label"`
	Depth         uint8          `json:"depth"`
	Amount        *bigint.BigInt `json:"amount"`
	BucketDepth   uint8          `json:"bucketDepth"`
	BlockNumber   uint64         `json:"blockNumber"`
	ImmutableFlag bool           `json:"immutableFlag"`
	Exists        bool           `json:"exists"`
	BatchTTL      int64          `json:"batchTTL"`
}

type postageStampsResponse struct {
	Stamps []PostageStampResponse `json:"stamps"`
}

// Sends a create postage request to a node that returns the batchID
func (p *PostageService) CreatePostageBatch(ctx context.Context, amount int64, depth uint64, gasPrice, label string) (batchID string, err error) {
	url := fmt.Sprintf("/stamps/%d/%d?label=%s", amount, depth, label)
	var resp postageResponse
	if gasPrice != "" {
		h := http.Header{}
		h.Add("Gas-Price", gasPrice)
		err = p.client.requestWithHeader(ctx, http.MethodPost, url, h, nil, &resp)
	} else {
		err = p.client.request(ctx, http.MethodPost, url, nil, &resp)
	}

	if err != nil {
		return "", err
	}
	return resp.BatchID, err
}

// Sends a topup batch request to a node that returns the batchID
func (p *PostageService) TopUpPostageBatch(ctx context.Context, batchID string, amount int64, gasPrice string) (err error) {
	url := fmt.Sprintf("/stamps/topup/%s/%d", batchID, amount)
	if gasPrice != "" {
		h := http.Header{}
		h.Add("Gas-Price", gasPrice)
		return p.client.requestWithHeader(ctx, http.MethodPatch, url, h, nil, nil)
	}
	return p.client.request(ctx, http.MethodPatch, url, nil, nil)
}

// Sends a dilute batch request to a node that returns the batchID
func (p *PostageService) DilutePostageBatch(ctx context.Context, batchID string, newDepth uint64, gasPrice string) (err error) {
	url := fmt.Sprintf("/stamps/dilute/%s/%d", batchID, newDepth)
	if gasPrice != "" {
		h := http.Header{}
		h.Add("Gas-Price", gasPrice)
		return p.client.requestWithHeader(ctx, http.MethodPatch, url, h, nil, nil)
	}
	return p.client.request(ctx, http.MethodPatch, url, nil, nil)
}

// Fetches the list postage stamp batches
func (p *PostageService) PostageBatches(ctx context.Context) ([]PostageStampResponse, error) {
	var resp postageStampsResponse
	err := p.client.request(ctx, http.MethodGet, "/stamps", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Stamps, nil
}

func (p *PostageService) PostageBatch(ctx context.Context, batchID string) (PostageStampResponse, error) {
	var resp PostageStampResponse
	err := p.client.request(ctx, http.MethodGet, "/stamps/"+batchID, nil, &resp)
	if err != nil {
		return PostageStampResponse{}, err
	}
	return resp, nil
}

type ReserveState struct {
	Radius        uint8          `json:"radius"`
	StorageRadius uint8          `json:"storageRadius"`
	Available     int64          `json:"available"`
	Outer         *bigint.BigInt `json:"outer"`
	Inner         *bigint.BigInt `json:"inner"`
}

func (rs ReserveState) String() string {
	return fmt.Sprintf("Radius: %d, StorageRadius: %d, Available: %d, Outer: %v, Inner: %v", rs.Radius, rs.StorageRadius, rs.Available, rs.Outer, rs.Inner)
}

// Returns the batchstore reservestate of the node
func (p *PostageService) ReserveState(ctx context.Context) (ReserveState, error) {
	var resp ReserveState
	err := p.client.request(ctx, http.MethodGet, "/reservestate", nil, &resp)
	return resp, err
}

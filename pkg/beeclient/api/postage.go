package api

import (
	"context"
	"fmt"
	"net/http"
)

// PostageService represents Bee's Postage service
type PostageService service

type postageResponse struct {
	BatchID string `json:"batchID"`
}

type PostageStampResponse struct {
	BatchID     string `json:"batchID"`
	Utilization uint32 `json:"utilization"`
}

type postageStampsResponse struct {
	Stamps []PostageStampResponse `json:"stamps"`
}

// Sends a create postage request to a node that returns the bactchID
func (p *PostageService) CreatePostageBatch(ctx context.Context, amount int64, depth uint64, label string) (string, error) {
	url := fmt.Sprintf("/%s/stamps/%d/%d?label=%s", apiVersion, amount, depth, label)
	var resp postageResponse
	err := p.client.request(ctx, http.MethodPost, url, nil, &resp)
	if err != nil {
		return "", err
	}
	return resp.BatchID, err
}

// Fetches the list postage stamp batches
func (p *PostageService) PostageBatches(ctx context.Context) ([]PostageStampResponse, error) {
	var resp postageStampsResponse
	err := p.client.request(ctx, http.MethodGet, fmt.Sprintf("/%s/stamps", apiVersion), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Stamps, nil
}

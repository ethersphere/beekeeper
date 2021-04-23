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

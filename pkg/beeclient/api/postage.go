package api

import (
	"context"
	"fmt"
	"net/http"
)

// PostageService represents Bee's Postage service
type PostageService service

type postageResponse struct {
	BatchID []byte
}

// Sends a create postage request to a node that returns the bactchID
func (p *PostageService) CreatePostageBatch(ctx context.Context, amount int, depth uint64, label string) (*postageResponse, error) {
	url := fmt.Sprintf("/%s/stamps/%d/%d?label=%s", apiVersion, amount, depth, label)
	var resp postageResponse
	return &resp, p.client.request(ctx, http.MethodPost, url, nil, &resp)
}

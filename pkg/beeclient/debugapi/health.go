package debugapi

import (
	"context"
	"net/http"
)

// HealthService represents Bee's health service
type HealthService service

// Check fetches health status of the instance
func (b *HealthService) Check(ctx context.Context, securityToken string) (resp HealthResponse, err error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Authorization", "Bearer "+securityToken)

	err = b.client.requestWithHeader(ctx, http.MethodGet, "/health", header, nil, &resp)

	return
}

// HealthResponse represents Bee's health response
type HealthResponse struct {
	Status string `json:"status"`
}

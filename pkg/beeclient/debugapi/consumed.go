package debugapi

import (
	"context"
	"net/http"
)

// ConsumedService represents Bee's balances service
type ConsumedService service

// Balances fetches balances from the instance
func (b *ConsumedService) Balances(ctx context.Context, securityToken string) (resp Balances, err error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Authorization", "Bearer "+securityToken)

	err = b.client.requestWithHeader(ctx, http.MethodGet, "/consumed", header, nil, &resp)

	return
}

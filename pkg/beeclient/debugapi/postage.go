package debugapi

import (
	"context"
	"math/big"
	"net/http"
)

type PostageService service

type ReserveState struct {
	Radius    uint8    `json:"radius"`
	Available int64    `json:"available"`
	Outer     *big.Int `json:"outer"`
	Inner     *big.Int `json:"inner"`
}

// Returns the batchstore reservestate of the node
func (p *PostageService) Reservestate(ctx context.Context) (ReserveState, error) {
	var resp ReserveState
	err := p.client.request(ctx, http.MethodGet, "/reservestate", nil, &resp)
	return resp, err
}

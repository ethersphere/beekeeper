package debugapi

import (
	"context"
	"github.com/ethersphere/beekeeper/pkg/bigint"
	"net/http"
)

type PostageService service

type ReserveState struct {
	Radius    uint8          `json:"radius"`
	Available int64          `json:"available"`
	Outer     *bigint.BigInt `json:"outer"`
	Inner     *bigint.BigInt `json:"inner"`
}

// Returns the batchstore reservestate of the node
func (p *PostageService) Reservestate(ctx context.Context) (ReserveState, error) {
	var resp ReserveState
	err := p.client.request(ctx, http.MethodGet, "/reservestate", nil, &resp)
	return resp, err
}

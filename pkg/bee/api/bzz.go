package api

import (
	"context"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// BzzService ...
type BzzService service

// BzzUploadResponse ...
type BzzUploadResponse struct {
	Hash swarm.Address `json:"hash"`
}

// Upload ...
func (b *BzzService) Upload(ctx context.Context, data io.Reader) (resp BzzUploadResponse, err error) {
	err = b.client.request(ctx, http.MethodPost, "/bzz", data, &resp)
	return
}

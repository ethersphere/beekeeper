package api

import (
	"context"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// BzzService represents Bee's Bzz service
type BzzService service

// UploadResponse represents Upload's response
type UploadResponse struct {
	Hash swarm.Address `json:"hash"`
}

// Upload uploads data to the node
func (b *BzzService) Upload(ctx context.Context, data io.Reader) (resp UploadResponse, err error) {
	err = b.client.request(ctx, http.MethodPost, "/bzz", data, &resp)
	return
}

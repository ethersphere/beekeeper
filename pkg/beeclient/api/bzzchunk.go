package api

import (
	"context"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// BzzChunkService represents Bee's Bzz service
type BzzChunkService service

// Download downloads data from the node
func (b *BzzChunkService) Download(ctx context.Context, a swarm.Address) (resp io.ReadCloser, err error) {
	return b.client.requestData(ctx, http.MethodGet, "/bzz-chunk/"+a.String(), nil, nil)
}

// BzzChunkUploadResponse represents Upload's response
type BzzChunkUploadResponse struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// Upload uploads data to the node
func (b *BzzChunkService) Upload(ctx context.Context, a swarm.Address, data io.Reader) (resp BzzChunkUploadResponse, err error) {
	err = b.client.request(ctx, http.MethodPost, "/bzz-chunk/"+a.String(), data, &resp)
	return
}

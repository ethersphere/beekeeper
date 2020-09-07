package api

import (
	"context"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// BytesService represents Bee's Bytes service
type BytesService service

// Download downloads data from the node
func (b *BytesService) Download(ctx context.Context, a swarm.Address) (resp io.ReadCloser, err error) {
	return b.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/bytes/"+a.String(), nil, nil)
}

// BytesUploadResponse represents Upload's response
type BytesUploadResponse struct {
	Reference swarm.Address `json:"reference"`
}

// Upload uploads bytes to the node
func (b *BytesService) Upload(ctx context.Context, data io.Reader) (resp BytesUploadResponse, err error) {
	err = b.client.requestJSON(ctx, http.MethodPost, "/"+apiVersion+"/bytes", data, &resp)
	return
}

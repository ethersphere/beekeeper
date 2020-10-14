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
func (b *BytesService) Upload(ctx context.Context, data io.Reader, pin bool) (resp BytesUploadResponse, err error) {
	h := http.Header{}
	if pin {
		h.Add("Swarm-Pin", "true")
	}

	err = b.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/bytes", h, data, &resp)
	return
}

package api

import (
	"context"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// ChunksService represents Bee's Chunks service
type ChunksService service

// Download downloads data from the node
func (c *ChunksService) Download(ctx context.Context, a swarm.Address, targets string) (resp io.ReadCloser, err error) {
	if targets == "" {
		return c.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/chunks/"+a.String(), nil, nil)
	}

	return c.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/chunks/"+a.String()+"?targets="+targets, nil, nil)
}

// ChunksUploadResponse represents Upload's response
type ChunksUploadResponse struct {
	Reference swarm.Address `json:"reference"`
}

// Upload uploads chunks to the node
func (c *ChunksService) Upload(ctx context.Context, a swarm.Address, data io.Reader, o UploadOptions) (ChunksUploadResponse, error) {
	var resp ChunksUploadResponse
	h := http.Header{}
	if o.Pin {
		h.Add("Swarm-Pin", "true")
	}
	err := c.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/chunks", h, data, &resp)
	return resp, err
}

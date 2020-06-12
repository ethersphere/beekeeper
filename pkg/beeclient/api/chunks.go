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
func (c *ChunksService) Download(ctx context.Context, a swarm.Address) (resp io.ReadCloser, err error) {
	return c.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/chunks/"+a.String(), nil, nil)
}

// ChunksUploadResponse represents Upload's response
type ChunksUploadResponse struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// Upload uploads chunks to the node
func (c *ChunksService) Upload(ctx context.Context, a swarm.Address, data io.Reader) (resp ChunksUploadResponse, err error) {
	err = c.client.request(ctx, http.MethodPost, "/"+apiVersion+"/chunks/"+a.String(), data, &resp)
	return
}

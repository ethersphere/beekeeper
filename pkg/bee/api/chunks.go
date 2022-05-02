package api

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// ChunksService represents Bee's Chunks service
type ChunksService service

// Remove removed the chunk from the node's local store
func (c *ChunksService) Remove(ctx context.Context, a swarm.Address) (err error) {
	err = c.client.request(ctx, http.MethodDelete, "/chunks/"+a.String(), nil, nil)
	return
}

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
func (c *ChunksService) Upload(ctx context.Context, data []byte, o UploadOptions) (ChunksUploadResponse, error) {
	var resp ChunksUploadResponse
	h := http.Header{}
	if o.Pin {
		h.Add("Swarm-Pin", "true")
	}
	if o.Deferred {
		h.Add("Swarm-Deferred-Upload", "true")
	}
	h.Add(postageStampBatchHeader, o.BatchID)
	err := c.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/chunks", h, bytes.NewReader(data), &resp)
	return resp, err
}

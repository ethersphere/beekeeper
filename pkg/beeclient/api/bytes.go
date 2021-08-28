package api

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/ethersphere/bee/pkg/swarm"
)

// BytesService represents Bee's Bytes service
type BytesService service

// Download downloads data from the node
func (b *BytesService) Download(ctx context.Context, a swarm.Address) (resp io.ReadCloser, err error) {
	r, err := b.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/bytes/"+a.String(), nil, nil, nil)

	return r.Body, err
}

// BytesUploadResponse represents Upload's response
type BytesUploadResponse struct {
	Reference swarm.Address `json:"reference"`
}

// Upload uploads bytes to the node
func (b *BytesService) Upload(ctx context.Context, data io.Reader, o UploadOptions) (BytesUploadResponse, error) {
	var resp BytesUploadResponse
	h := http.Header{}
	if o.Pin {
		h.Add("Swarm-Pin", "true")
	}
	if o.Tag != 0 {
		h.Add("Swarm-Tag", strconv.FormatUint(uint64(o.Tag), 10))
	}
	h.Add(postageStampBatchHeader, o.BatchID)
	err := b.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/bytes", h, data, &resp)
	return resp, err
}

package api

import (
	"context"
	"io"
	"net/http"
)

// BzzService ...
type BzzService service

// BzzGetResponse ...
type BzzGetResponse struct {
	Hash string `json:"hash"`
}

// Get ...
func (b *BzzService) Get(ctx context.Context, addr string) (resp BzzGetResponse, err error) {
	var r BzzGetResponse
	err = b.client.request(ctx, http.MethodPost, "/bzz/"+addr, nil, &r)
	return r, err
}

// BzzUploadResponse ...
type BzzUploadResponse struct {
	Hash string `json:"hash"`
}

// Upload ...
func (b *BzzService) Upload(ctx context.Context, data io.Reader) (resp BzzUploadResponse, err error) {
	var r BzzUploadResponse
	err = b.client.request(ctx, http.MethodPost, "/bzz/", data, &r)
	return r, err
}

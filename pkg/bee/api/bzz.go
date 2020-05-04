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

// BzzGet ...
func (b *BzzService) BzzGet(ctx context.Context, addr string) (resp BzzGetResponse, err error) {
	var r BzzGetResponse
	err = b.client.request(ctx, http.MethodPost, "/bzz/"+addr, nil, &r)
	return r, err
}

// BzzUploadResponse ...
type BzzUploadResponse struct {
	Hash string `json:"hash"`
}

// BzzUpload ...
func (b *BzzService) BzzUpload(ctx context.Context, data io.Reader) (resp BzzUploadResponse, err error) {
	var r BzzUploadResponse
	err = b.client.request(ctx, http.MethodPost, "/bzz/", data, &r)
	return r, err
}

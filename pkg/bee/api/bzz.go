package api

import (
	"context"
	"io"
	"net/http"
)

// BzzService ...
type BzzService service

// UploadResponse ...
type UploadResponse struct {
	Hash string `json:"hash"`
}

// Upload ...
func (b *BzzService) Upload(ctx context.Context, chunk io.Reader) (uploadResponse UploadResponse, err error) {
	var r UploadResponse
	err = b.client.request(ctx, http.MethodPost, "/bzz/", chunk, &r)
	return r, err
}

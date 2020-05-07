package api

import (
	"context"
	"io"
	"net/http"
)

// BzzService ...
type BzzService service

// BzzUploadResponse ...
type BzzUploadResponse struct {
	Hash string `json:"hash"`
}

// Upload ...
func (b *BzzService) Upload(ctx context.Context, data io.Reader) (resp BzzUploadResponse, err error) {
	err = b.client.request(ctx, http.MethodPost, "/bzz", data, &resp)
	return
}

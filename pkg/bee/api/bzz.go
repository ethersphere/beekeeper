package api

import (
	"context"
	"net/http"
)

// BzzService ...
type BzzService service

// BzzUploadResponse ...
type BzzUploadResponse struct {
	Hash string `json:"hash"`
}

// Upload ...
func (b *BzzService) Upload(ctx context.Context, data interface{}) (resp BzzUploadResponse, err error) {
	var r BzzUploadResponse
	err = b.client.request(ctx, http.MethodPost, "/bzz", data, &r)
	return r, err
}

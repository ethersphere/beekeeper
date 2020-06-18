package api

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/ethersphere/bee/pkg/swarm"
)

// FilesService represents Bee's Files service
type FilesService service

// Download downloads data from the node
func (f *FilesService) Download(ctx context.Context, a swarm.Address) (resp io.ReadCloser, err error) {
	return f.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/files/"+a.String(), nil, nil)
}

// FilesUploadResponse represents Upload's response
type FilesUploadResponse struct {
	Reference swarm.Address `json:"reference"`
}

// Upload uploads files to the node
func (f *FilesService) Upload(ctx context.Context, name, contentType string, contentLength int, data io.Reader) (resp FilesUploadResponse, err error) {
	header := make(http.Header)
	header.Set("Content-Type", contentType)
	header.Set("Content-Length", strconv.Itoa(contentLength))
	// TODO: escape name
	err = f.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/files"+"?name="+name, header, data, &resp)
	return
}

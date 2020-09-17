package api

import (
	"context"
	"io"
	"net/http"
	"net/url"
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
func (f *FilesService) Upload(ctx context.Context, name string, data io.Reader, size int64, pin bool) (resp FilesUploadResponse, err error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/octet-stream")
	header.Set("Content-Length", strconv.FormatInt(size, 10))
	if pin {
		header.Set("Swarm-Pin", "true")
	}

	err = f.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/files?"+url.QueryEscape("name="+name), header, data, &resp)
	return
}

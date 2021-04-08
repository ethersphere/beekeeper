package api

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/ethersphere/bee/pkg/swarm"
)

// DirsService represents Bee's Dirs service
type DirsService service

// Download downloads data from the node
func (s *DirsService) Download(ctx context.Context, a swarm.Address, path string) (resp io.ReadCloser, err error) {
	return s.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/bzz/"+a.String()+"/"+path, nil, nil)
}

// DirsUploadResponse represents Upload's response
type DirsUploadResponse struct {
	Reference swarm.Address `json:"reference"`
}

// Upload uploads TAR collection to the node
func (s *DirsService) Upload(ctx context.Context, data io.Reader, size int64) (resp DirsUploadResponse, err error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/x-tar")
	header.Set("Content-Length", strconv.FormatInt(size, 10))

	err = s.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/dirs", header, data, &resp)

	return
}

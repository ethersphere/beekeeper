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
func (s *DirsService) Upload(ctx context.Context, data io.Reader, size int64, o UploadOptions) (resp DirsUploadResponse, err error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/x-tar")
	header.Set("Content-Length", strconv.FormatInt(size, 10))
	header.Set("swarm-collection", "True")
	header.Set(postageStampBatchHeader, o.BatchID)

	err = s.client.RequestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/bzz", header, data, &resp)

	return
}

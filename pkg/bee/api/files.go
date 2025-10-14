package api

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// FilesService represents Bee's Files service
type FilesService service

// Download downloads data from the node
func (f *FilesService) Download(ctx context.Context, a swarm.Address, opts *DownloadOptions) (resp io.ReadCloser, err error) {
	return f.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/bzz/"+a.String(), nil, opts)
}

// FilesUploadResponse represents Upload's response
type FilesUploadResponse struct {
	Reference swarm.Address `json:"reference"`
}

// Upload uploads files to the node
func (f *FilesService) Upload(ctx context.Context, name string, data io.Reader, size int64, o UploadOptions) (resp FilesUploadResponse, err error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/octet-stream")
	header.Add("Content-Length", strconv.FormatInt(size, 10))
	if o.Pin {
		header.Add(swarmPinHeader, "true")
	}
	if o.Tag != 0 {
		header.Add(swarmTagHeader, strconv.FormatUint(o.Tag, 10))
	}
	if o.Direct {
		header.Add(deferredUploadHeader, strconv.FormatBool(false))
	}
	header.Add(postageStampBatchHeader, o.BatchID)

	err = f.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/bzz?"+url.QueryEscape("name="+name), header, data, &resp)
	return resp, err
}

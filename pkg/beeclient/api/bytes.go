package api

import (
	"context"
	"io"
	"net/http"
	"time"
	"fmt"

	"github.com/ethersphere/bee/pkg/swarm"
)


const (
	maxAttemptsAfterSent  = 10
)

// BytesService represents Bee's Bytes service
type BytesService service

// Download downloads data from the node
func (b *BytesService) Download(ctx context.Context, a swarm.Address) (resp io.ReadCloser, err error) {
	return b.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/bytes/"+a.String(), nil, nil)
}

// BytesUploadResponse represents Upload's response
type BytesUploadResponse struct {
	Reference swarm.Address `json:"reference"`
}

// Upload uploads bytes to the node
func (b *BytesService) Upload(ctx context.Context, data io.Reader, o UploadOptions) (BytesUploadResponse, error) {
	var resp BytesUploadResponse
	h := http.Header{}
	if o.Pin {
		h.Add("Swarm-Pin", "true")
	}
	_, err := b.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/bytes", h, data, &resp)
	return resp, err
}

// Upload uploads bytes to the node
func (b *BytesService) UploadAndSync(ctx context.Context, data io.Reader, o UploadOptions) (BytesUploadResponse, error) {
	var resp BytesUploadResponse
	h := http.Header{}
	if o.Pin {
		h.Add("Swarm-Pin", "true")
	}

	r, err := b.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/bytes", h, data, &resp)

	tag := r.Header["Swarm-Tag"][0]

	var tr TagResponse
	err = b.client.requestJSON(ctx, http.MethodGet, "/tags/"+tag, nil, &tr)

	var lastSynced int64
	attemptAfterSent := 0
	syncing := true
	for syncing == true {

		fmt.Println(tr)

		if tr.Synced >= tr.Total{
			syncing = false
		}
		lastSynced = tr.Synced

		time.Sleep(1000 * time.Millisecond)

		if lastSynced == tr.Synced {
			attemptAfterSent++
		}else{
			attemptAfterSent = 0
		}

		if attemptAfterSent > maxAttemptsAfterSent {
			syncing = false
		}
	}

	return resp, err
}


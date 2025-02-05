package api

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

type ActService service

type ActUploadResponse struct {
	Reference      swarm.Address `json:"reference"`
	HistoryAddress swarm.Address
}

type ActGranteesResponse struct {
	Reference      swarm.Address `json:"ref"`
	HistoryAddress swarm.Address `json:"historyref"`
}

func (a *ActService) Download(ctx context.Context, addr swarm.Address, opts *DownloadOptions) (resp io.ReadCloser, err error) {
	return a.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/bzz/"+addr.String()+"/", nil, opts)
}

func (a *ActService) Upload(ctx context.Context, name string, data io.Reader, o UploadOptions) (ActUploadResponse, error) {
	var resp ActUploadResponse
	h := http.Header{}
	h.Add(postageStampBatchHeader, o.BatchID)
	h.Add("swarm-deferred-upload", "true")
	h.Add("content-type", "application/octet-stream")
	h.Add("Swarm-Act", "true")
	h.Add(swarmPinHeader, "true")
	historyParser := func(h http.Header) {
		resp.HistoryAddress, _ = swarm.ParseHexAddress(h.Get("Swarm-Act-History-Address"))
	}
	err := a.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/bzz?"+url.QueryEscape("name="+name), h, data, &resp, historyParser)
	return resp, err
}

func (a *ActService) AddGrantees(ctx context.Context, data io.Reader, o UploadOptions) (ActGranteesResponse, error) {
	var resp ActGranteesResponse
	h := http.Header{}
	h.Add(postageStampBatchHeader, o.BatchID)
	h.Add(swarmActHistoryAddress, o.ActHistoryAddress.String())
	err := a.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/grantee", h, data, &resp)
	return resp, err
}

func (a *ActService) GetGrantees(ctx context.Context, addr swarm.Address) (resp io.ReadCloser, err error) {
	return a.client.requestData(ctx, http.MethodGet, "/"+apiVersion+"/grantee/"+addr.String(), nil, nil)
}

func (a *ActService) PatchGrantees(ctx context.Context, data io.Reader, addr swarm.Address, haddr swarm.Address, batchID string) (ActGranteesResponse, error) {
	var resp ActGranteesResponse
	h := http.Header{}
	h.Add("swarm-postage-batch-id", batchID)
	h.Add("swarm-act-history-address", haddr.String())
	err := a.client.requestWithHeader(ctx, http.MethodPatch, "/"+apiVersion+"/grantee/"+addr.String(), h, data, &resp)
	return resp, err
}

package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// SOCService represents Bee's SOC service.
type SOCService service

// SocResponse is the JSON body returned by POST /soc.
type SocResponse struct {
	Reference swarm.Address `json:"reference"`
}

// UploadSOC uploads a single owner chunk using a postage batch ID.
func (p *SOCService) UploadSOC(ctx context.Context, owner, ID, signature string, data io.Reader, batchID string) (*SocResponse, error) {
	h := http.Header{}
	h.Add(postageStampBatchHeader, batchID)
	h.Add(deferredUploadHeader, "false")
	url := fmt.Sprintf("/%s/soc/%s/%s?sig=%s", apiVersion, owner, ID, signature)

	resp := SocResponse{}
	return &resp, p.client.requestWithHeader(ctx, http.MethodPost, url, h, data, &resp)
}

// UploadSOCWithStamp uploads a single owner chunk using a precomputed postage stamp
// (hex-encoded Swarm-Postage-Stamp header). Direct upload is forced so the chunk
// enters the reserve immediately for pullsync.
func (p *SOCService) UploadSOCWithStamp(ctx context.Context, owner, ID, signature string, data io.Reader, stampHex string) (*SocResponse, error) {
	h := http.Header{}
	h.Add(postageStampHeader, stampHex)
	h.Add(deferredUploadHeader, "false")
	url := fmt.Sprintf("/%s/soc/%s/%s?sig=%s", apiVersion, owner, ID, signature)

	resp := SocResponse{}
	return &resp, p.client.requestWithHeader(ctx, http.MethodPost, url, h, data, &resp)
}

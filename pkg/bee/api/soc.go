package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// PSSService represents Bee's PSS service
type SOCService service

type SOCResponse struct {
	Reference swarm.Address
}

type SOCOptions struct {
	RLevel *redundancy.Level
}

// Sends a PSS message to a recipienct with a specific topic
func (p *SOCService) UploadSOC(ctx context.Context, owner, ID, signature string, data io.Reader, batchID string, opt *SOCOptions) (*SOCResponse, error) {
	h := http.Header{}
	h.Add(postageStampBatchHeader, batchID)
	if opt != nil && opt.RLevel != nil {
		h.Add(SwarmRedundancyLevelHeader, fmt.Sprintf("%d", *opt.RLevel))
	}
	url := fmt.Sprintf("/%s/soc/%s/%s?sig=%s", apiVersion, owner, ID, signature)

	resp := SOCResponse{}
	return &resp, p.client.requestWithHeader(ctx, http.MethodPost, url, h, data, &resp)
}

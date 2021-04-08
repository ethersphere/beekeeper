package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// PSSService represents Bee's PSS service
type SOCService service

type SocResponse struct {
	Reference swarm.Address
}

// Sends a PSS message to a recipienct with a specific topic
func (p *SOCService) UploadSOC(ctx context.Context, owner, ID, signature string, data io.Reader) (*SocResponse, error) {

	url := fmt.Sprintf("/%s/soc/%s/%s?sig=%s", apiVersion, owner, ID, signature)

	resp := SocResponse{}
	return &resp, p.client.request(ctx, http.MethodPost, url, data, &resp)
}

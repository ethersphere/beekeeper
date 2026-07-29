package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/ethersphere/bee/v2/pkg/postage"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// EnvelopeService represents Bee's postage envelope service.
type EnvelopeService service

// EnvelopeResponse is the JSON body returned by POST /envelope/{address}.
type EnvelopeResponse struct {
	Issuer    string `json:"issuer"`
	Index     string `json:"index"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
}

// Create requests a postage stamp for the given chunk address from the node's
// stamper for batchID. The stamp is returned as a postage.Stamp ready for upload.
func (e *EnvelopeService) Create(ctx context.Context, addr swarm.Address, batchID string) (*postage.Stamp, error) {
	h := http.Header{}
	h.Add(postageStampBatchHeader, batchID)

	var resp EnvelopeResponse
	err := e.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/envelope/"+addr.String(), h, nil, &resp)
	if err != nil {
		return nil, err
	}

	batchBytes, err := hex.DecodeString(batchID)
	if err != nil {
		return nil, fmt.Errorf("decode batch id: %w", err)
	}
	index, err := hex.DecodeString(resp.Index)
	if err != nil {
		return nil, fmt.Errorf("decode stamp index: %w", err)
	}
	ts, err := hex.DecodeString(resp.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("decode stamp timestamp: %w", err)
	}
	sig, err := hex.DecodeString(resp.Signature)
	if err != nil {
		return nil, fmt.Errorf("decode stamp signature: %w", err)
	}
	return postage.NewStamp(batchBytes, index, ts, sig), nil
}

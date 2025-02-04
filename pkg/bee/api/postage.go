package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bigint"
)

// PostageService represents Bee's Postage service
type PostageService service

type postageResponse struct {
	BatchID string `json:"batchID"`
}

type PostageStampResponse struct {
	BatchID       string         `json:"batchID"`
	Utilization   uint32         `json:"utilization"`
	Usable        bool           `json:"usable"`
	Label         string         `json:"label"`
	Depth         uint8          `json:"depth"`
	Amount        *bigint.BigInt `json:"amount"`
	BucketDepth   uint8          `json:"bucketDepth"`
	BlockNumber   uint64         `json:"blockNumber"`
	ImmutableFlag bool           `json:"immutableFlag"`
	Exists        bool           `json:"exists"`
	BatchTTL      int64          `json:"batchTTL"`
}

type postageStampsResponse struct {
	Stamps []PostageStampResponse `json:"stamps"`
}

// Sends a create postage request to a node that returns the batchID
func (p *PostageService) CreatePostageBatch(ctx context.Context, amount int64, depth uint64, label string) (batchID string, err error) {
	url := fmt.Sprintf("/stamps/%d/%d?label=%s", amount, depth, label)
	var resp postageResponse

	h := http.Header{}
	h.Add("Immutable", "false")

	if err := p.client.requestWithHeader(ctx, http.MethodPost, url, h, nil, &resp); err != nil {
		return "", err
	}

	return resp.BatchID, nil
}

// Sends a topup batch request to a node that returns the batchID
func (p *PostageService) TopUpPostageBatch(ctx context.Context, batchID string, amount int64, gasPrice string) (err error) {
	url := fmt.Sprintf("/stamps/topup/%s/%d", url.PathEscape(batchID), amount)
	if gasPrice != "" {
		h := http.Header{}
		h.Add("Gas-Price", gasPrice)
		return p.client.requestWithHeader(ctx, http.MethodPatch, url, h, nil, nil)
	}
	return p.client.request(ctx, http.MethodPatch, url, nil, nil)
}

// Sends a dilute batch request to a node that returns the batchID
func (p *PostageService) DilutePostageBatch(ctx context.Context, batchID string, newDepth uint64, gasPrice string) (err error) {
	url := fmt.Sprintf("/stamps/dilute/%s/%d", url.PathEscape(batchID), newDepth)
	if gasPrice != "" {
		h := http.Header{}
		h.Add("Gas-Price", gasPrice)
		return p.client.requestWithHeader(ctx, http.MethodPatch, url, h, nil, nil)
	}
	return p.client.request(ctx, http.MethodPatch, url, nil, nil)
}

// Fetches the list postage stamp batches
func (p *PostageService) PostageBatches(ctx context.Context) ([]PostageStampResponse, error) {
	var resp postageStampsResponse

	if err := p.client.request(ctx, http.MethodGet, "/stamps", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Stamps, nil
}

func (p *PostageService) PostageStamp(ctx context.Context, batchID string) (PostageStampResponse, error) {
	var resp PostageStampResponse

	if err := p.client.request(ctx, http.MethodGet, "/stamps/"+batchID, nil, &resp); err != nil {
		return PostageStampResponse{}, err
	}
	return resp, nil
}

type ReserveState struct {
	Radius        uint8 `json:"radius"`
	StorageRadius uint8 `json:"storageRadius"`
}

func (rs ReserveState) String() string {
	return fmt.Sprintf("Radius: %d, StorageRadius: %d", rs.Radius, rs.StorageRadius)
}

// Returns the batchstore reservestate of the node
func (p *PostageService) ReserveState(ctx context.Context) (ReserveState, error) {
	var resp ReserveState
	err := p.client.request(ctx, http.MethodGet, "/reservestate", nil, &resp)
	return resp, err
}

type ChainStateResponse struct {
	ChainTip     uint64         `json:"chainTip"`     // ChainTip (block height).
	Block        uint64         `json:"block"`        // The block number of the last postage event.
	TotalAmount  *bigint.BigInt `json:"totalAmount"`  // Cumulative amount paid per stamp. //*big.Int
	CurrentPrice *bigint.BigInt `json:"currentPrice"` // Bzz/chunk/block normalised price. //*big.Int
}

// GetChainState returns the chain state of the node
func (p *PostageService) GetChainState(ctx context.Context) (ChainStateResponse, error) {
	var resp ChainStateResponse

	if err := p.client.request(ctx, http.MethodGet, "/chainstate", nil, &resp); err != nil {
		return ChainStateResponse{}, err
	}

	return resp, nil
}

func (batch *PostageStampResponse) BatchUsage() float64 {
	maxUtilization := 1 << (batch.Depth - batch.BucketDepth)            // 2^(depth - bucketDepth)
	return (float64(batch.Utilization) / float64(maxUtilization)) * 100 // batch utilization between 0 and 100 percent
}

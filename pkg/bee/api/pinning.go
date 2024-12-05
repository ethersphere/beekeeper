package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// PinningService represents Bee's Pin service
type PinningService service

// pinsBasePath is the pins API base path for http requests.
const pinsBasePath = "/pins"

func pinsPath(path string) string { return pinsBasePath + "/" + path }

// PinRootHash pins root hash of given reference.
func (ps *PinningService) PinRootHash(ctx context.Context, ref swarm.Address) error {
	res := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}
	return ps.client.requestJSON(ctx, http.MethodPost, pinsPath(ref.String()), nil, &res)
}

// UnpinRootHash unpins root hash of given reference.
func (ps *PinningService) UnpinRootHash(ctx context.Context, ref swarm.Address) error {
	res := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}
	return ps.client.requestJSON(ctx, http.MethodDelete, pinsPath(ref.String()), nil, &res)
}

// GetPinnedRootHash determines if the root hash of
// given reference is pinned by returning its reference.
func (ps *PinningService) GetPinnedRootHash(ctx context.Context, ref swarm.Address) (swarm.Address, error) {
	res := struct {
		Reference swarm.Address `json:"reference"`
	}{}
	err := ps.client.requestJSON(ctx, http.MethodGet, pinsPath(ref.String()), nil, &res)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("get pinned root hash: %w", err)
	}
	return res.Reference, nil
}

// GetPins returns all references of pinned root hashes.
func (ps *PinningService) GetPins(ctx context.Context) ([]swarm.Address, error) {
	res := struct {
		References []swarm.Address `json:"references"`
	}{}
	err := ps.client.requestJSON(ctx, http.MethodGet, pinsBasePath, nil, &res)
	if err != nil {
		return nil, fmt.Errorf("get pins: %w", err)
	}
	return res.References, nil
}

package api

import (
	"context"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// PinningService represents Bee's Pin service
type PinningService service

// PinChunk pins chunk
func (p *PinningService) PinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	err := p.client.requestJSON(ctx, http.MethodPost, "/pinning/chunks/"+a.String(), nil, &resp)
	if err == ErrNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// PinnedChunk represents pinned chunk
type PinnedChunk struct {
	Address    swarm.Address `json:"address"`
	PinCounter int           `json:"pinCounter"`
}

// PinnedChunk gets pinned chunk
func (p *PinningService) PinnedChunk(ctx context.Context, a swarm.Address) (resp PinnedChunk, err error) {
	err = p.client.requestJSON(ctx, http.MethodGet, "/pinning/chunks/"+a.String(), nil, &resp)
	return
}

// PinnedChunks represents pinned chunks
type PinnedChunks struct {
	Chunks []PinnedChunk `json:"chunks"`
}

// PinnedChunks gets pinned chunks
func (p *PinningService) PinnedChunks(ctx context.Context) (resp PinnedChunks, err error) {
	err = p.client.requestJSON(ctx, http.MethodGet, "/pinning/chunks", nil, &resp)
	return
}

// UnpinChunk unpins chunk
func (p *PinningService) UnpinChunk(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodDelete, "/pinning/chunks/"+a.String(), nil, &resp)
}

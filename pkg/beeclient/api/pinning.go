package api

import (
	"context"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// PinningService represents Bee's Pin service
type PinningService service

// PinChunk pins chunk
func (p *PinningService) PinChunk(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodPost, "/pin/chunks/"+a.String(), nil, &resp)
}

// PinnedChunk represents pinned chunk
type PinnedChunk struct {
	Address    swarm.Address `json:"address"`
	PinCounter int           `json:"pinCounter"`
}

// PinnedChunk gets pinned chunk
func (p *PinningService) PinnedChunk(ctx context.Context, a swarm.Address) (resp PinnedChunk, err error) {
	err = p.client.requestJSON(ctx, http.MethodGet, "/pin/chunks/"+a.String(), nil, &resp)
	return
}

// PinnedChunks represents pinned chunks
type PinnedChunks struct {
	Chunks []PinnedChunk `json:"chunks"`
}

// PinnedChunks gets pinned chunks
func (p *PinningService) PinnedChunks(ctx context.Context) (resp PinnedChunks, err error) {
	err = p.client.requestJSON(ctx, http.MethodGet, "/pin/chunks", nil, &resp)
	return
}

// UnpinChunk unpins chunk
func (p *PinningService) UnpinChunk(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodDelete, "/pin/chunks/"+a.String(), nil, &resp)
}

// PinBytes pins chunks for bytes upload
func (p *PinningService) PinBytes(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodPost, "/pin/bytes/"+a.String(), nil, &resp)
}

// UnpinBytes unpins chunks for bytes upload
func (p *PinningService) UnpinBytes(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodDelete, "/pin/bytes/"+a.String(), nil, &resp)
}

// PinFiles pins chunks for files upload
func (p *PinningService) PinFiles(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodPost, "/pin/files/"+a.String(), nil, &resp)
}

// UnpinFiles unpins chunks for files upload
func (p *PinningService) UnpinFiles(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodDelete, "/pin/files/"+a.String(), nil, &resp)
}

// PinBzz pins chunks for bzz upload
func (p *PinningService) PinBzz(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodPost, "/pin/bzz/"+a.String(), nil, &resp)
}

// UnpinBzz unpins chunks for bzz upload
func (p *PinningService) UnpinBzz(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return p.client.requestJSON(ctx, http.MethodDelete, "/pin/bzz/"+a.String(), nil, &resp)
}

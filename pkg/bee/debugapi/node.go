package debugapi

import (
	"context"
	"net/http"
)

// NodeService ...
type NodeService service

// Addresses ...
type Addresses struct {
	Overlay  string   `json:"overlay"`
	Underlay []string `json:"underlay"`
}

// Addresses ...
func (n *NodeService) Addresses(ctx context.Context) (resp Addresses, err error) {
	var r Addresses
	err = n.client.request(ctx, http.MethodGet, "/addresses", nil, &r)
	return r, err
}

// StatusResponse ...
type StatusResponse struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// HasChunk ...
func (n *NodeService) HasChunk(ctx context.Context, address string) (resp StatusResponse, err error) {
	var r StatusResponse
	err = n.client.request(ctx, http.MethodGet, "/chunks/"+address, nil, &r)
	return r, err
}

// Peers ...
type Peers struct {
	Peers []Peer `json:"peers"`
}

// Peer ...
type Peer struct {
	Address string `json:"address"`
}

// Peers ...
func (n *NodeService) Peers(ctx context.Context) (resp Peers, err error) {
	var r Peers
	err = n.client.request(ctx, http.MethodGet, "/peers", nil, &r)
	return r, err
}

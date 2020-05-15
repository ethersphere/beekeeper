package debugapi

import (
	"context"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// NodeService ...
type NodeService service

// Addresses ...
type Addresses struct {
	Overlay  swarm.Address `json:"overlay"`
	Underlay []string      `json:"underlay"`
}

// Addresses ...
func (n *NodeService) Addresses(ctx context.Context) (resp Addresses, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/addresses", nil, &resp)
	return
}

// StatusResponse ...
type StatusResponse struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// HasChunk ...
func (n *NodeService) HasChunk(ctx context.Context, address swarm.Address) (resp StatusResponse, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/chunks/"+address.String(), nil, &resp)
	return
}

// Peers ...
type Peers struct {
	Peers []Peer `json:"peers"`
}

// Peer ...
type Peer struct {
	Address swarm.Address `json:"address"`
}

// Peers ...
func (n *NodeService) Peers(ctx context.Context) (resp Peers, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/peers", nil, &resp)
	return
}

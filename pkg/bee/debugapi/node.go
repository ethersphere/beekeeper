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
func (n *NodeService) Addresses(ctx context.Context) (addresses Addresses, err error) {
	var r Addresses
	err = n.client.request(ctx, http.MethodGet, "/addresses", nil, &r)
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
func (n *NodeService) Peers(ctx context.Context) (peers Peers, err error) {
	var r Peers
	err = n.client.request(ctx, http.MethodGet, "/peers", nil, &r)
	return r, err
}

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
func (n *NodeService) Addresses(ctx context.Context) (a Addresses, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/addresses", nil, &a)
	return
}

// HasChunk ...
func (n *NodeService) HasChunk(ctx context.Context, address swarm.Address) (bool, error) {
	r := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	err := n.client.requestJSON(ctx, http.MethodGet, "/chunks/"+address.String(), nil, &r)
	if err == ErrNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
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
func (n *NodeService) Peers(ctx context.Context) (p Peers, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/peers", nil, &p)
	return
}

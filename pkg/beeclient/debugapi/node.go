package debugapi

import (
	"context"
	"net/http"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
)

// NodeService represents Bee's Node service
type NodeService service

// Addresses represents node's addresses
type Addresses struct {
	Overlay  swarm.Address `json:"overlay"`
	Underlay []string      `json:"underlay"`
}

// Addresses returns node's addresses
func (n *NodeService) Addresses(ctx context.Context) (resp Addresses, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/addresses", nil, &resp)
	return
}

// HasChunk returns true/false if node has a chunk
func (n *NodeService) HasChunk(ctx context.Context, a swarm.Address) (bool, error) {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	err := n.client.requestJSON(ctx, http.MethodGet, "/chunks/"+a.String(), nil, &resp)
	if err == ErrNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// Health represents node's health
type Health struct {
	Status string `json:"status"`
}

// Health returns node's health
func (n *NodeService) Health(ctx context.Context) (resp Health, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/health", nil, &resp)
	return
}

// Peers represents node's peers
type Peers struct {
	Peers []Peer `json:"peers"`
}

// Peer represents node's peer
type Peer struct {
	Address swarm.Address `json:"address"`
}

// Peers returns node's peers
func (n *NodeService) Peers(ctx context.Context) (resp Peers, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/peers", nil, &resp)
	return
}

// PinChunk pins chunk
func (n *NodeService) PinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	err := n.client.requestJSON(ctx, http.MethodPost, "/chunks-pin/"+a.String(), nil, &resp)
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
func (n *NodeService) PinnedChunk(ctx context.Context, a swarm.Address) (resp PinnedChunk, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/chunks-pin/"+a.String(), nil, &resp)
	return
}

// PinnedChunks represents pinned chunks
type PinnedChunks struct {
	Chunks []PinnedChunk `json:"chunks"`
}

// PinnedChunks gets pinned chunks
func (n *NodeService) PinnedChunks(ctx context.Context) (resp PinnedChunks, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/chunks-pin", nil, &resp)
	return
}

// Readiness represents node's readiness
type Readiness struct {
	Status string `json:"status"`
}

// Readiness returns node's readiness
func (n *NodeService) Readiness(ctx context.Context) (resp Readiness, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/readiness", nil, &resp)
	return
}

// Topology represents Kademlia topology
type Topology struct {
	BaseAddr       swarm.Address  `json:"baseAddr"`
	Population     int            `json:"population"`
	Connected      int            `json:"connected"`
	Timestamp      time.Time      `json:"timestamp"`
	NnLowWatermark int            `json:"nnLowWatermark"`
	Depth          int            `json:"depth"`
	Bins           map[string]Bin `json:"bins"`
}

// Bin represents Kademlia bin
type Bin struct {
	Population        int             `json:"population"`
	Connected         int             `json:"connected"`
	DisconnectedPeers []swarm.Address `json:"disconnectedPeers"`
	ConnectedPeers    []swarm.Address `json:"connectedPeers"`
}

// Topology returns Kademlia topology
func (n *NodeService) Topology(ctx context.Context) (resp Topology, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/topology", nil, &resp)
	if err != nil {
		return Topology{}, err
	}

	return
}

// UnpinChunk unpins chunk
func (n *NodeService) UnpinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	err := n.client.requestJSON(ctx, http.MethodDelete, "/chunks-pin/"+a.String(), nil, &resp)
	if err == ErrNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

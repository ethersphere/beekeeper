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
	Ethereum  string        `json:"ethereum"`
	Overlay   swarm.Address `json:"overlay"`
	PublicKey string        `json:"public_key"`
	Underlay  []string      `json:"underlay"`
}

// Addresses returns node's addresses
func (n *NodeService) Addresses(ctx context.Context) (resp Addresses, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/addresses", nil, &resp)
	return
}

// Balance represents node's balance with a peer
type Balance struct {
	Balance int64  `json:"balance"`
	Peer    string `json:"peer"`
}

// Balance returns node's balance with a given peer
func (n *NodeService) Balance(ctx context.Context, a swarm.Address) (resp Balance, err error) {
	err = n.client.request(ctx, http.MethodGet, "/balances/"+a.String(), nil, &resp)
	return
}

// Balances represents node's balances with all peers
type Balances struct {
	Balances []Balance `json:"balances"`
}

// Balances returns node's balances with all peers
func (n *NodeService) Balances(ctx context.Context) (resp Balances, err error) {
	err = n.client.request(ctx, http.MethodGet, "/balances", nil, &resp)
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

// RemoveChunk removes chunk from the node
func (n *NodeService) RemoveChunk(ctx context.Context, a swarm.Address) error {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	return n.client.requestJSON(ctx, http.MethodDelete, "/chunks/"+a.String(), nil, &resp)
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

// Readiness represents node's readiness
type Readiness struct {
	Status string `json:"status"`
}

// Readiness returns node's readiness
func (n *NodeService) Readiness(ctx context.Context) (resp Readiness, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/readiness", nil, &resp)
	return
}

// Settlement represents node's settlement with a peer
type Settlement struct {
	Peer     string `json:"peer"`
	Received int    `json:"received"`
	Sent     int    `json:"sent"`
}

// Settlement returns node's settlement with a given peer
func (n *NodeService) Settlement(ctx context.Context, a swarm.Address) (resp Settlement, err error) {
	err = n.client.request(ctx, http.MethodGet, "/settlements/"+a.String(), nil, &resp)
	return
}

// Settlements represents node's settlements with all peers
type Settlements struct {
	Settlements   []Settlement `json:"settlements"`
	TotalReceived int          `json:"totalreceived"`
	TotalSent     int          `json:"totalsent"`
}

// Settlements returns node's settlements with all peers
func (n *NodeService) Settlements(ctx context.Context) (resp Settlements, err error) {
	err = n.client.request(ctx, http.MethodGet, "/settlements", nil, &resp)
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
	LightNodes     Bin            `json:"lightNodes"`
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

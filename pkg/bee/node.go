package bee

import (
	"bytes"
	"context"
	"net/url"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
)

// Node represents Bee node
type Node struct {
	api   *api.Client
	debug *debugapi.Client
}

// NodeOptions represents Bee node options
type NodeOptions struct {
	APIURL   *url.URL
	DebugURL *url.URL
}

// NewNode returns new node
func NewNode(opts NodeOptions) Node {
	return Node{
		api:   api.NewClient(opts.APIURL, nil),
		debug: debugapi.NewClient(opts.DebugURL, nil),
	}
}

// HasChunk returns true/false if node has a chunk
func (n *Node) HasChunk(ctx context.Context, c Chunk) (bool, error) {
	return n.debug.Node.HasChunk(ctx, c.Address())
}

// Overlay returns node's overlay address
func (n *Node) Overlay(ctx context.Context) (swarm.Address, error) {
	a, err := n.debug.Node.Addresses(ctx)
	if err != nil {
		return swarm.Address{}, err
	}

	return a.Overlay, nil
}

// Peers returns addresses of node's peers
func (n *Node) Peers(ctx context.Context) (peers []swarm.Address, err error) {
	ps, err := n.debug.Node.Peers(ctx)
	if err != nil {
		return []swarm.Address{}, err
	}

	for _, p := range ps.Peers {
		peers = append(peers, p.Address)
	}

	return
}

// Ping pings other node
func (n *Node) Ping(ctx context.Context, node swarm.Address) (rtt string, err error) {
	r, err := n.api.PingPong.Ping(ctx, node)
	if err != nil {
		return "", err
	}
	return r.RTT, nil
}

// UploadChunk uploads chunk to the node
func (n *Node) UploadChunk(ctx context.Context, c *Chunk) (err error) {
	r, err := n.api.Bzz.Upload(ctx, bytes.NewReader(c.Data()))
	if err != nil {
		return err
	}

	c.address = r.Hash

	return
}

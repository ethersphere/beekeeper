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

	// TODO:
	// Chunks []Chunk
}

// NodeOptions represents Bee node options
type NodeOptions struct {
	APIURL   *url.URL
	DebugURL *url.URL
}

// NewNode returns Bee node
func NewNode(opts NodeOptions) Node {
	return Node{
		api:   api.NewClient(opts.APIURL, nil),
		debug: debugapi.NewClient(opts.DebugURL, nil),
	}
}

// Debug returns Bee debug API Client
func (n *Node) Debug() *debugapi.Client {
	return n.debug
}

// Ping pings other Bee node
func (n *Node) Ping(ctx context.Context, node swarm.Address) (rtt string, err error) {
	r, err := n.api.PingPong.Ping(ctx, node)
	if err != nil {
		return "", err
	}
	return r.RTT, nil
}

// UploadChunk uploads chunk to the node
func (n *Node) UploadChunk(ctx context.Context, c Chunk) (address swarm.Address, err error) {
	r, err := n.api.Bzz.Upload(ctx, bytes.NewReader(c.Data()))
	if err != nil {
		return swarm.Address{}, err
	}
	return r.Hash, nil
}

package bee

import (
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

// Node represents Bee node
type Node struct {
	name         string
	client       *Client
	config       *k8sBee.Config
	clefKey      string
	clefPassword string
	libP2PKey    string
	swarmKey     string
}

// NodeOptions holds optional parameters for the Node.
type NodeOptions struct {
	Client       *Client
	Config       *k8sBee.Config
	ClefKey      string
	ClefPassword string
	LibP2PKey    string
	SwarmKey     string
}

// NewNode returns Bee node
func NewNode(name string, opts NodeOptions) (n *Node) {
	n = &Node{name: name}

	if opts.Client != nil {
		n.client = opts.Client
	}
	if opts.Config != nil {
		n.config = opts.Config
	}
	if len(opts.ClefKey) > 0 {
		n.clefKey = opts.ClefKey
	}
	if len(opts.ClefPassword) > 0 {
		n.clefPassword = opts.ClefPassword
	}
	if len(opts.LibP2PKey) > 0 {
		n.libP2PKey = opts.LibP2PKey
	}
	if len(opts.SwarmKey) > 0 {
		n.swarmKey = opts.SwarmKey
	}

	return
}

// Name returns node's name
func (n *Node) Name() string {
	return n.name
}

// Client returns node's name
func (n *Node) Client() *Client {
	return n.client
}

// Config returns node's config
func (n *Node) Config() *k8sBee.Config {
	return n.config
}

// ClefKey returns node's clefKey
func (n *Node) ClefKey() string {
	return n.clefKey
}

// ClefPassword returns node's clefPassword
func (n *Node) ClefPassword() string {
	return n.clefPassword
}

// LibP2PKey returns node's libP2PKey
func (n *Node) LibP2PKey() string {
	return n.libP2PKey
}

// SwarmKey returns node's swarmKey
func (n *Node) SwarmKey() string {
	return n.swarmKey
}

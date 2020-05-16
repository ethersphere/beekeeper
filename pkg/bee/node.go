package bee

import (
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
)

// Node represents Bee node
type Node struct {
	api      *api.Client
	debugAPI *debugapi.Client
}

// NodeOptions represents Bee node options
type NodeOptions struct {
	APIURL      *url.URL
	DebugAPIURL *url.URL
}

// NewNode returns Bee node
func NewNode(opts NodeOptions) Node {
	return Node{
		api:      api.NewClient(opts.APIURL, nil),
		debugAPI: debugapi.NewClient(opts.DebugAPIURL, nil),
	}
}

// API returns Bee API Client
func (n *Node) API() *api.Client {
	return n.api
}

// DebugAPI returns Bee debug API Client
func (n *Node) DebugAPI() *debugapi.Client {
	return n.debugAPI
}

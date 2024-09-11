package k8s

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// compile check whether client implements interface
var _ orchestration.Node = (*Node)(nil)

// Node represents Bee node
type Node struct {
	orchestration.NodeOrchestrator
	name string
	opts orchestration.NodeOptions
	log  logging.Logger
}

// NewNode returns Bee node
func NewNode(name string, opts orchestration.NodeOptions, no orchestration.NodeOrchestrator, log logging.Logger) (n *Node) {
	return &Node{
		NodeOrchestrator: no,
		name:             name,
		opts:             opts,
		log:              log,
	}
}

// Name returns node's name
func (n Node) Name() string {
	return n.name
}

// Client returns node's name
func (n Node) Client() *bee.Client {
	return n.opts.Client
}

// Config returns node's config
func (n Node) Config() *orchestration.Config {
	return n.opts.Config
}

// LibP2PKey returns node's libP2PKey
func (n Node) LibP2PKey() string {
	return n.opts.LibP2PKey
}

// SwarmKey returns node's swarmKey
func (n Node) SwarmKey() string {
	return n.opts.SwarmKey.String()
}

// SetSwarmKey sets node's Swarm key
func (n Node) SetSwarmKey(key string) orchestration.Node {
	n.opts.SwarmKey = orchestration.EncryptedKey(key)
	return n
}

// Create implements orchestration.Node.
// Subtle: this method shadows the method (NodeOrchestrator).Create of Node.NodeOrchestrator.
func (n Node) Create(ctx context.Context, o orchestration.CreateOptions) (err error) {
	return n.NodeOrchestrator.Create(ctx, o)
}

// Delete implements orchestration.Node.
// Subtle: this method shadows the method (NodeOrchestrator).Delete of Node.NodeOrchestrator.
func (n Node) Delete(ctx context.Context, namespace string) (err error) {
	return n.NodeOrchestrator.Delete(ctx, n.name, namespace)
}

// Ready implements orchestration.Node.
// Subtle: this method shadows the method (NodeOrchestrator).Ready of Node.NodeOrchestrator.
func (n Node) Ready(ctx context.Context, namespace string) (ready bool, err error) {
	return n.NodeOrchestrator.Ready(ctx, n.name, namespace)
}

// Start implements orchestration.Node.
// Subtle: this method shadows the method (NodeOrchestrator).Start of Node.NodeOrchestrator.
func (n Node) Start(ctx context.Context, namespace string) (err error) {
	return n.NodeOrchestrator.Start(ctx, n.name, namespace)
}

// Stop implements orchestration.Node.
// Subtle: this method shadows the method (NodeOrchestrator).Stop of Node.NodeOrchestrator.
func (n Node) Stop(ctx context.Context, namespace string) (err error) {
	return n.NodeOrchestrator.Stop(ctx, n.name, namespace)
}

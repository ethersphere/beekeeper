package nokube

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
	name string
	opts orchestration.NodeOptions
	log  logging.Logger
}

// NewNode returns Bee node
func NewNode(name string, opts orchestration.NodeOptions, log logging.Logger) (n *Node) {
	return &Node{
		name: name,
		opts: opts,
		log:  log,
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

// ClefKey returns node's clefKey
func (n Node) ClefKey() string {
	return n.opts.ClefKey
}

// ClefPassword returns node's clefPassword
func (n Node) ClefPassword() string {
	return n.opts.ClefPassword
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

// SetClefKey sets node's Clef key
func (n Node) SetClefKey(key string) orchestration.Node {
	n.opts.ClefKey = key
	return n
}

// SetClefKey sets node's Clef key
func (n Node) SetClefPassword(password string) orchestration.Node {
	n.opts.ClefPassword = password
	return n
}

func (n Node) Create(ctx context.Context, o orchestration.CreateOptions) (err error) {
	return orchestration.ErrNotSet
}

func (n Node) Delete(ctx context.Context, namespace string) (err error) {
	return orchestration.ErrNotSet
}

func (n Node) Ready(ctx context.Context, namespace string) (ready bool, err error) {
	return false, orchestration.ErrNotSet
}

func (n Node) Start(ctx context.Context, namespace string) (err error) {
	return orchestration.ErrNotSet
}

func (n Node) Stop(ctx context.Context, namespace string) (err error) {
	return orchestration.ErrNotSet
}

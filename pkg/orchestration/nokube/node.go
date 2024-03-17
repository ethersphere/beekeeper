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
	name         string
	clefKey      string
	clefPassword string
	client       *bee.Client
	config       *orchestration.Config
	libP2PKey    string
	swarmKey     string
	logger       logging.Logger
}

// NewNode returns a new Bee node configured with the provided options and logger.
func NewNode(name string, opts orchestration.NodeOptions, logger logging.Logger) *Node {
	return &Node{
		name:         name,
		client:       opts.Client,
		config:       opts.Config,
		clefKey:      opts.ClefKey,
		clefPassword: opts.ClefPassword,
		libP2PKey:    opts.LibP2PKey,
		swarmKey:     opts.SwarmKey.String(),
		logger:       logger,
	}
}

// Name returns node's name
func (n Node) Name() string {
	return n.name
}

// Client returns node's name
func (n Node) Client() *bee.Client {
	return n.client
}

// Config returns node's config
func (n Node) Config() *orchestration.Config {
	return n.config
}

// ClefKey returns node's clefKey
func (n Node) ClefKey() string {
	return n.clefKey
}

// ClefPassword returns node's clefPassword
func (n Node) ClefPassword() string {
	return n.clefPassword
}

// LibP2PKey returns node's libP2PKey
func (n Node) LibP2PKey() string {
	return n.libP2PKey
}

// SwarmKey returns node's swarmKey
func (n Node) SwarmKey() string {
	return n.swarmKey
}

// SetSwarmKey sets node's Swarm key
func (n Node) SetSwarmKey(key string) orchestration.Node {
	n.swarmKey = key
	return n
}

// SetClefKey sets node's Clef key
func (n Node) SetClefKey(key string) orchestration.Node {
	n.clefKey = key
	return n
}

// SetClefKey sets node's Clef key
func (n Node) SetClefPassword(password string) orchestration.Node {
	n.clefPassword = password
	return n
}

func (n Node) Create(ctx context.Context, o orchestration.CreateOptions) (err error) {
	panic("unimplemented")
}

func (n Node) Delete(ctx context.Context, namespace string) (err error) {
	panic("unimplemented")
}

func (n Node) Ready(ctx context.Context, namespace string) (ready bool, err error) {
	panic("unimplemented")
}

func (n Node) Start(ctx context.Context, namespace string) (err error) {
	panic("unimplemented")
}

func (n Node) Stop(ctx context.Context, namespace string) (err error) {
	panic("unimplemented")
}

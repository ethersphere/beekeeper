package orchestration

import "github.com/ethersphere/beekeeper/pkg/bee"

type Node interface {
	Name() string
	Client() *bee.Client
	Config() *Config
	ClefKey() string
	ClefPassword() string
	LibP2PKey() string
	SwarmKey() string
}

// NodeOptions holds optional parameters for the Node.
type NodeOptions struct {
	ClefKey      string
	ClefPassword string
	Client       *bee.Client
	Config       *Config
	LibP2PKey    string
	SwarmKey     string
}

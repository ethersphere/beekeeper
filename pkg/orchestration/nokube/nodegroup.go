package nokube

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
)

// compile check whether client implements interface
var _ orchestration.NodeGroup = (*NodeGroup)(nil)

// NodeGroup represents group of Bee nodes
type NodeGroup struct {
	*k8s.NodeGroup
	clusterOpts orchestration.ClusterOptions
	opts        orchestration.NodeGroupOptions
	log         logging.Logger
}

// NewNodeGroup returns new node group
func NewNodeGroup(name string, o orchestration.NodeGroupOptions, co orchestration.ClusterOptions, log logging.Logger) *NodeGroup {
	return &NodeGroup{
		NodeGroup:   k8s.NewNodeGroup(name, o, co, log),
		clusterOpts: co,
		opts:        o,
		log:         log,
	}
}

// AddNode adss new node to the node group
func (g *NodeGroup) AddNode(ctx context.Context, name string, o orchestration.NodeOptions) (err error) {
	aURL, err := g.clusterOpts.ApiURL(name)
	if err != nil {
		return fmt.Errorf("API URL %s: %w", name, err)
	}

	dURL, err := g.clusterOpts.DebugAPIURL(name)
	if err != nil {
		return fmt.Errorf("debug API URL %s: %w", name, err)
	}

	// TODO: make more granular, check every sub-option
	var config *orchestration.Config
	if o.Config != nil {
		config = o.Config
	} else {
		config = g.opts.BeeConfig
	}

	client := bee.NewClient(bee.ClientOptions{
		APIURL:              aURL,
		APIInsecureTLS:      g.clusterOpts.APIInsecureTLS,
		DebugAPIURL:         dURL,
		DebugAPIInsecureTLS: g.clusterOpts.DebugAPIInsecureTLS,
		Retry:               5,
		Restricted:          config.Restricted,
	}, g.log)

	n := NewNode(name, orchestration.NodeOptions{
		ClefKey:      o.ClefKey,
		ClefPassword: o.ClefPassword,
		Client:       client,
		Config:       config,
		LibP2PKey:    o.LibP2PKey,
		SwarmKey:     o.SwarmKey,
	}, g.log)

	g.Lock.Lock()
	g.Nodes[n.Name()] = n
	g.Lock.Unlock()

	return
}

// RunningNodes returns list of running nodes
func (g *NodeGroup) RunningNodes(ctx context.Context) (running []string, err error) {
	return g.NodesSorted(), nil
}

// StoppedNodes returns list of stopped nodes
func (g *NodeGroup) StoppedNodes(ctx context.Context) (stopped []string, err error) {
	return stopped, nil
}

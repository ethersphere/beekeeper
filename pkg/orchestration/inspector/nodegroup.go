package inspector

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
)

// compile check whether client implements interface
var _ orchestration.NodeGroup = (*NodeGroup)(nil)

// NodeGroup represents group of Bee nodes
type NodeGroup struct {
	*k8s.NodeGroup
}

// NewNodeGroup returns new node group
func NewNodeGroup(name string, o orchestration.NodeGroupOptions, logger logging.Logger) *NodeGroup {
	return &NodeGroup{
		NodeGroup: k8s.NewNodeGroup(name, o, logger),
	}
}

// RunningNodes returns list of running nodes
func (g *NodeGroup) RunningNodes(ctx context.Context) (running []string, err error) {
	return g.NodesSorted(), nil
}

// StoppedNodes returns list of stopped nodes
func (g *NodeGroup) StoppedNodes(ctx context.Context) (stopped []string, err error) {
	return stopped, nil
}

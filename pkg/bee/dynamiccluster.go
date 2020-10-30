package bee

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

// DynamicCluster ...
type DynamicCluster struct {
	k8s        *k8s.Client
	k8sBee     *k8sBee.Client
	name       string
	nodeGroups map[string]NodeGroup
}

// DynamicClusterOptions ...
type DynamicClusterOptions struct {
	k8s  *k8sBee.Client
	name string
}

// NewDynamicCluster ...
func NewDynamicCluster(o DynamicClusterOptions) *DynamicCluster {
	return &DynamicCluster{
		k8sBee: o.k8s,
		name:   o.name,
	}
}

// NodeGroup ...
type NodeGroup struct {
	Nodes   map[string]Client
	Options NodeGroupOptions
}

// NodeGroupOptions ...
type NodeGroupOptions struct {
	name string
}

// Start starts cluster with given options
func (dc *DynamicCluster) Start(ctx context.Context) (err error) {
	// dc.k8s.Namespace.Create()
	return
}

// NewNodeGroup ...
func (dc *DynamicCluster) NewNodeGroup(o NodeGroupOptions) {
	dc.nodeGroups[o.name] = NodeGroup{
		Nodes:   make(map[string]Client),
		Options: o,
	}
}

// NodeStartOptions ...
type NodeStartOptions struct {
	GroupName string
	Config    k8sBee.Config
}

// NodeStart ...
func (dc *DynamicCluster) NodeStart(o NodeStartOptions) (err error) {
	ctx := context.Background()
	return dc.k8sBee.NodeStart(ctx, k8sBee.NodeStartOptions{})
}

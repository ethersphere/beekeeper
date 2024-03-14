package nokube

import (
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
)

// compile check whether client implements interface
var _ orchestration.Cluster = (*Cluster)(nil)

// Cluster represents cluster of Bee nodes
type Cluster struct {
	*k8s.Cluster

	log  logging.Logger
	opts orchestration.ClusterOptions
}

// NewCluster returns new cluster
func NewCluster(name string, o orchestration.ClusterOptions, log logging.Logger) *Cluster {
	return &Cluster{
		Cluster: k8s.NewCluster(name, o, log),
		log:     log,
		opts:    o,
	}
}

// AddNodeGroup adds new node group to the cluster
func (c *Cluster) AddNodeGroup(name string, o orchestration.NodeGroupOptions) {
	g := NewNodeGroup(name, o, c.opts, c.log)
	c.NodeGroups[name] = g
}

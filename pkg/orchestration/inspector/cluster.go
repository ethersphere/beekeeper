package inspector

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
}

// NewCluster returns new cluster
func NewCluster(name string, o orchestration.ClusterOptions, logger logging.Logger) *Cluster {
	return &Cluster{
		Cluster: k8s.NewCluster(name, o, logger),
	}
}

package orchestration

import (
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// NodeGroupOptions represents node group options
type NodeGroupOptions struct {
	Annotations               map[string]string
	ClefImage                 string
	ClefImagePullPolicy       string
	BeeConfig                 *Config
	Image                     string
	ImagePullPolicy           string
	ImagePullSecrets          []string
	IngressAnnotations        map[string]string
	IngressClass              string
	IngressDebugAnnotations   map[string]string
	IngressDebugClass         string
	Labels                    map[string]string
	NodeSelector              map[string]string
	PersistenceEnabled        bool
	PersistenceStorageClass   string
	PersistenceStorageRequest string
	PodManagementPolicy       string
	RestartPolicy             string
	ResourcesLimitCPU         string
	ResourcesLimitMemory      string
	ResourcesRequestCPU       string
	ResourcesRequestMemory    string
	UpdateStrategy            string
}

// NodeGroupAddresses represents addresses of all nodes in the node group
type NodeGroupAddresses map[string]bee.Addresses

// NodeGroupBalances represents balances of all nodes in the node group
type NodeGroupBalances map[string]map[string]int64

type FundingOptions struct {
	Eth  float64
	Bzz  float64
	GBzz float64
}

// NodeGroupOverlays represents overlay addresses of all nodes in the node group
type NodeGroupOverlays map[string]swarm.Address

// NodeGroupPeers represents peers of all nodes in the node group
type NodeGroupPeers map[string][]swarm.Address

// NodeGroupSettlements represents settlements of all nodes in the node group
type NodeGroupSettlements map[string]map[string]SentReceived

// SentReceived object
type SentReceived struct {
	Received int64
	Sent     int64
}

// NodeGroupTopologies represents Kademlia topology of all nodes in the node group
type NodeGroupTopologies map[string]bee.Topology

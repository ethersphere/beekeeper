package orchestration

import (
	"context"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/swap"
)

type Cluster interface {
	AddNodeGroup(name string, o NodeGroupOptions)
	Addresses(ctx context.Context) (addrs map[string]NodeGroupAddresses, err error)
	Balances(ctx context.Context) (balances ClusterBalances, err error)
	FlattenBalances(ctx context.Context) (balances NodeGroupBalances, err error)
	GlobalReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error)
	Name() string
	NodeGroups() (l map[string]NodeGroup)
	NodeGroupsSorted() (l []string)
	NodeGroup(name string) (ng NodeGroup, err error)
	Nodes() map[string]Node
	NodeNames() (names []string)
	LightNodeNames() (names []string)
	FullNodeNames() (names []string)
	NodesClients(ctx context.Context) (map[string]*bee.Client, error)
	NodesClientsAll(ctx context.Context) (map[string]*bee.Client, error)
	Overlays(ctx context.Context, exclude ...string) (overlays ClusterOverlays, err error)
	FlattenOverlays(ctx context.Context, exclude ...string) (map[string]swarm.Address, error)
	Peers(ctx context.Context, exclude ...string) (peers ClusterPeers, err error)
	RandomNode(ctx context.Context, r *rand.Rand) (node Node, err error)
	Settlements(ctx context.Context) (settlements ClusterSettlements, err error)
	FlattenSettlements(ctx context.Context) (settlements NodeGroupSettlements, err error)
	Size() (size int)
	Topologies(ctx context.Context) (topologies ClusterTopologies, err error)
	FlattenTopologies(ctx context.Context) (topologies map[string]bee.Topology, err error)
}

// ClusterOptions represents Bee cluster options
type ClusterOptions struct {
	Annotations         map[string]string
	APIDomain           string
	APIInsecureTLS      bool
	APIScheme           string
	DebugAPIDomain      string
	DebugAPIInsecureTLS bool
	DebugAPIScheme      string
	K8SClient           *k8s.Client
	SwapClient          swap.Client
	Labels              map[string]string
	Namespace           string
	DisableNamespace    bool
	AdminPassword       string
}

// ClusterAddresses represents addresses of all nodes in the cluster
type ClusterAddresses map[string]NodeGroupAddresses

// ClusterBalances represents balances of all nodes in the cluster
type ClusterBalances map[string]NodeGroupBalances

// ClusterOverlays represents overlay addresses of all nodes in the cluster
type ClusterOverlays map[string]NodeGroupOverlays

// ClusterPeers represents peers of all nodes in the cluster
type ClusterPeers map[string]NodeGroupPeers

// ClusterSettlements represents settlements of all nodes in the cluster
type ClusterSettlements map[string]NodeGroupSettlements

// ClusterTopologies represents Kademlia topology of all nodes in the cluster
type ClusterTopologies map[string]NodeGroupTopologies

// RandomOverlay returns a random overlay from a random NodeGroup
func (c ClusterOverlays) Random(r *rand.Rand) (nodeGroup string, nodeName string, overlay swarm.Address) {
	i := r.Intn(len(c))
	var (
		ng, name string
		ngo      NodeGroupOverlays
		o        swarm.Address
	)
	for n, v := range c {
		if i == 0 {
			ng = n
			ngo = v
			break
		}
		i--
	}

	i = r.Intn(len(ngo))

	for n, v := range ngo {
		if i == 0 {
			name = n
			o = v
			break
		}
		i--
	}
	return ng, name, o
}

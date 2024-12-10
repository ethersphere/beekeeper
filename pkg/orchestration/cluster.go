package orchestration

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/swap"
)

type Cluster interface {
	Accounting(ctx context.Context) (accounting ClusterAccounting, err error)
	AddNodeGroup(name string, o NodeGroupOptions)
	Addresses(ctx context.Context) (addrs map[string]NodeGroupAddresses, err error)
	Balances(ctx context.Context) (balances ClusterBalances, err error)
	FlattenAccounting(ctx context.Context) (accounting NodeGroupAccounting, err error)
	FlattenBalances(ctx context.Context) (balances NodeGroupBalances, err error)
	FlattenOverlays(ctx context.Context, exclude ...string) (map[string]swarm.Address, error)
	FlattenSettlements(ctx context.Context) (settlements NodeGroupSettlements, err error)
	FlattenTopologies(ctx context.Context) (topologies map[string]bee.Topology, err error)
	FullNodeNames() (names []string)
	GlobalReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error)
	LightNodeNames() (names []string)
	Name() string
	NodeGroup(name string) (ng NodeGroup, err error)
	NodeGroups() (l map[string]NodeGroup)
	NodeGroupsSorted() (l []string)
	NodeNames() (names []string)
	Nodes() map[string]Node
	NodesClients(ctx context.Context) (map[string]*bee.Client, error)
	NodesClientsAll(ctx context.Context) (map[string]*bee.Client, error)
	Overlays(ctx context.Context, exclude ...string) (overlays ClusterOverlays, err error)
	Peers(ctx context.Context, exclude ...string) (peers ClusterPeers, err error)
	RandomNode(ctx context.Context, r *rand.Rand) (node Node, err error)
	Settlements(ctx context.Context) (settlements ClusterSettlements, err error)
	Size() (size int)
	Topologies(ctx context.Context) (topologies ClusterTopologies, err error)
}

// ClusterOptions represents Bee cluster options
type ClusterOptions struct {
	Annotations       map[string]string
	APIDomain         string
	APIDomainInternal string
	APIInsecureTLS    bool
	APIScheme         string
	K8SClient         *k8s.Client
	SwapClient        swap.Client
	Labels            map[string]string
	Namespace         string
	DisableNamespace  bool
}

// ClusterAddresses represents addresses of all nodes in the cluster
type ClusterAddresses map[string]NodeGroupAddresses

// ClusterAccounting represents accounting of all nodes in the cluster
type ClusterAccounting map[string]NodeGroupAccounting

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

// ApiURL generates URL for node's API
func (c ClusterOptions) ApiURL(name string, inCluster bool) (u *url.URL, err error) {
	apiDomain := c.APIDomain
	apiScheme := c.APIScheme
	if inCluster {
		apiDomain = c.APIDomainInternal
		apiScheme = "http"
	}
	if c.DisableNamespace {
		u, err = url.Parse(fmt.Sprintf("%s://%s.%s", apiScheme, name, apiDomain))
	} else {
		u, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", apiScheme, name, c.Namespace, apiDomain))
	}
	if err != nil {
		return nil, fmt.Errorf("bad API url for node %s: %w", name, err)
	}
	return
}

// IngressHost generates host for node's API ingress
func (c ClusterOptions) IngressHost(name string) string {
	if c.DisableNamespace {
		return fmt.Sprintf("%s.%s", name, c.APIDomain)
	}
	return fmt.Sprintf("%s.%s.%s", name, c.Namespace, c.APIDomain)
}

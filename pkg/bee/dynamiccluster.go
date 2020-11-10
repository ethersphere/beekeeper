package bee

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

// DynamicCluster represents cluster of Bee nodes
type DynamicCluster struct {
	name                string
	annotations         map[string]string
	apiDomain           string
	apiInsecureTLS      bool
	apiScheme           string
	debugAPIDomain      string
	debugAPIInsecureTLS bool
	debugAPIScheme      string
	k8s                 *k8s.Client
	labels              map[string]string
	namespace           string
	disableNamespace    bool                  // do not use namespace for node hostnames
	nodeGroups          map[string]*NodeGroup // set when groups are added to the cluster
}

// DynamicClusterOptions represents Bee cluster options
type DynamicClusterOptions struct {
	Annotations         map[string]string
	APIDomain           string
	APIInsecureTLS      bool
	APIScheme           string
	DebugAPIDomain      string
	DebugAPIInsecureTLS bool
	DebugAPIScheme      string
	KubeconfigPath      string
	Labels              map[string]string
	Namespace           string
	DisableNamespace    bool
}

// NewDynamicCluster returns new cluster
func NewDynamicCluster(name string, o DynamicClusterOptions) *DynamicCluster {
	k8s := k8s.NewClient(&k8s.ClientOptions{KubeconfigPath: o.KubeconfigPath})

	return &DynamicCluster{
		name:                name,
		annotations:         o.Annotations,
		apiDomain:           o.APIDomain,
		apiInsecureTLS:      o.APIInsecureTLS,
		apiScheme:           o.APIScheme,
		debugAPIDomain:      o.DebugAPIDomain,
		debugAPIInsecureTLS: o.DebugAPIInsecureTLS,
		debugAPIScheme:      o.DebugAPIScheme,
		k8s:                 k8s,
		labels:              o.Labels,
		namespace:           o.Namespace,
		disableNamespace:    o.DisableNamespace,

		nodeGroups: make(map[string]*NodeGroup),
	}
}

// AddNodeGroup adds new node group to the cluster
func (dc *DynamicCluster) AddNodeGroup(name string, o NodeGroupOptions) {
	g := NewNodeGroup(name, o)
	g.cluster = dc
	g.k8s = k8sBee.NewClient(g.cluster.k8s)
	g.opts.Annotations = mergeMaps(g.cluster.annotations, o.Annotations)
	g.opts.Labels = mergeMaps(g.cluster.labels, o.Labels)

	dc.nodeGroups[name] = g
}

// ClusterAddresses represents addresses of all nodes in the cluster
type ClusterAddresses map[string]NodeGroupAddresses

// Addresses returns ClusterAddresses
func (dc *DynamicCluster) Addresses(ctx context.Context) (addrs map[string]NodeGroupAddresses, err error) {
	addrs = make(ClusterAddresses)

	for k, v := range dc.nodeGroups {
		a, err := v.Addresses(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		addrs[k] = a
	}

	return
}

// ClusterBalances represents balances of all nodes in the cluster
type ClusterBalances map[string]NodeGroupBalances

// Balances returns ClusterBalances
func (dc *DynamicCluster) Balances(ctx context.Context) (balances ClusterBalances, err error) {
	balances = make(ClusterBalances)

	for k, v := range dc.nodeGroups {
		b, err := v.Balances(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		balances[k] = b
	}

	return
}

// GlobalReplicationFactor returns the total number of nodes in the cluster that contain given chunk
func (dc *DynamicCluster) GlobalReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error) {
	for k, v := range dc.nodeGroups {
		ngrf, err := v.GroupReplicationFactor(ctx, a)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", k, err)
		}

		grf += ngrf
	}

	return
}

// Name returns name of the cluster
func (dc *DynamicCluster) Name() string {
	return dc.name
}

// NodeGroups returns map of node groups in the cluster
func (dc *DynamicCluster) NodeGroups() (l map[string]*NodeGroup) {
	return dc.nodeGroups
}

// NodeGroupsSorted returns sorted list of node names in the node group
func (dc *DynamicCluster) NodeGroupsSorted() (l []string) {
	for k := range dc.nodeGroups {
		l = append(l, k)
	}
	sort.Strings(l)
	return
}

// NodeGroup returns node group
func (dc *DynamicCluster) NodeGroup(name string) *NodeGroup {
	return dc.nodeGroups[name]
}

// ClusterOverlays represents overlay addresses of all nodes in the cluster
type ClusterOverlays map[string]NodeGroupOverlays

// Overlays returns ClusterOverlays
func (dc *DynamicCluster) Overlays(ctx context.Context) (overlays ClusterOverlays, err error) {
	overlays = make(ClusterOverlays)

	for k, v := range dc.nodeGroups {
		o, err := v.Overlays(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		overlays[k] = o
	}

	return
}

// ClusterPeers represents peers of all nodes in the cluster
type ClusterPeers map[string]NodeGroupPeers

// Peers returns peers of all nodes in the cluster
func (dc *DynamicCluster) Peers(ctx context.Context) (peers ClusterPeers, err error) {
	peers = make(ClusterPeers)

	for k, v := range dc.nodeGroups {
		p, err := v.Peers(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		peers[k] = p
	}

	return
}

// ClusterSettlements represents settlements of all nodes in the cluster
type ClusterSettlements map[string]NodeGroupSettlements

// Settlements returns
func (dc *DynamicCluster) Settlements(ctx context.Context) (settlements ClusterSettlements, err error) {
	settlements = make(ClusterSettlements)

	for k, v := range dc.nodeGroups {
		s, err := v.Settlements(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		settlements[k] = s
	}

	return
}

// Size returns size of the cluster
func (dc *DynamicCluster) Size() (size int) {
	for _, ng := range dc.nodeGroups {
		size += len(ng.nodes)
	}
	return
}

// ClusterTopologies represents Kademlia topology of all nodes in the cluster
type ClusterTopologies map[string]NodeGroupTopologies

// Topologies returns ClusterTopologies
func (dc *DynamicCluster) Topologies(ctx context.Context) (topologies ClusterTopologies, err error) {
	topologies = make(ClusterTopologies)

	for k, v := range dc.nodeGroups {
		t, err := v.Topologies(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		topologies[k] = t
	}

	return
}

// apiURL generates URL for node's API
func (dc *DynamicCluster) apiURL(name string) (u *url.URL, err error) {
	if dc.disableNamespace {
		u, err = url.Parse(fmt.Sprintf("%s://%s.%s", dc.apiScheme, name, dc.apiDomain))
	} else {
		u, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", dc.apiScheme, name, dc.namespace, dc.apiDomain))
	}
	if err != nil {
		return nil, fmt.Errorf("bad API url for node %s: %s", name, err)
	}
	return
}

// ingressHost generates host for node's API ingress
func (dc *DynamicCluster) ingressHost(name string) string {
	if dc.disableNamespace {
		return fmt.Sprintf("%s.%s", name, dc.apiDomain)
	}
	return fmt.Sprintf("%s.%s.%s", name, dc.namespace, dc.apiDomain)
}

// debugAPIURL generates URL for node's DebugAPI
func (dc *DynamicCluster) debugAPIURL(name string) (u *url.URL, err error) {
	if dc.disableNamespace {
		u, err = url.Parse(fmt.Sprintf("%s://%s-debug.%s", dc.debugAPIScheme, name, dc.debugAPIDomain))
	} else {
		u, err = url.Parse(fmt.Sprintf("%s://%s-debug.%s.%s", dc.debugAPIScheme, name, dc.namespace, dc.debugAPIDomain))
	}
	if err != nil {
		return nil, fmt.Errorf("bad debug API url for node %s: %s", name, err)
	}
	return
}

// ingressHost generates host for node's DebugAPI ingress
func (dc *DynamicCluster) ingressDebugHost(name string) string {
	if dc.disableNamespace {
		return fmt.Sprintf("%s-debug.%s", name, dc.debugAPIDomain)
	}
	return fmt.Sprintf("%s-debug.%s.%s", name, dc.namespace, dc.debugAPIDomain)
}

// mergeMaps joins two maps
func mergeMaps(a, b map[string]string) map[string]string {
	m := map[string]string{}
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}

	return m
}

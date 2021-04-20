package bee

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"sort"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s/notset"
)

// Cluster represents cluster of Bee nodes
type Cluster struct {
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
	Labels              map[string]string
	Namespace           string
	DisableNamespace    bool
}

// NewCluster returns new cluster
func NewCluster(name string, o ClusterOptions) *Cluster {
	return &Cluster{
		name:                name,
		annotations:         o.Annotations,
		apiDomain:           o.APIDomain,
		apiInsecureTLS:      o.APIInsecureTLS,
		apiScheme:           o.APIScheme,
		debugAPIDomain:      o.DebugAPIDomain,
		debugAPIInsecureTLS: o.DebugAPIInsecureTLS,
		debugAPIScheme:      o.DebugAPIScheme,
		k8s:                 o.K8SClient,
		labels:              o.Labels,
		namespace:           o.Namespace,
		disableNamespace:    o.DisableNamespace,

		nodeGroups: make(map[string]*NodeGroup),
	}
}

// AddNodeGroup adds new node group to the cluster
func (c *Cluster) AddNodeGroup(name string, o NodeGroupOptions) {
	g := NewNodeGroup(name, o)
	g.cluster = c

	if g.cluster.k8s != nil {
		g.k8s = k8sBee.NewClient(g.cluster.k8s)
	} else {
		g.k8s = new(notset.BeeClient)
	}

	g.opts.Annotations = mergeMaps(g.cluster.annotations, o.Annotations)
	g.opts.Labels = mergeMaps(g.cluster.labels, o.Labels)

	c.nodeGroups[name] = g
}

// ClusterAddresses represents addresses of all nodes in the cluster
type ClusterAddresses map[string]NodeGroupAddresses

// Addresses returns ClusterAddresses
func (c *Cluster) Addresses(ctx context.Context) (addrs map[string]NodeGroupAddresses, err error) {
	addrs = make(ClusterAddresses)

	for k, v := range c.nodeGroups {
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
func (c *Cluster) Balances(ctx context.Context) (balances ClusterBalances, err error) {
	balances = make(ClusterBalances)

	for k, v := range c.nodeGroups {
		b, err := v.Balances(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		balances[k] = b
	}

	return
}

// GlobalReplicationFactor returns the total number of nodes in the cluster that contain given chunk
func (c *Cluster) GlobalReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error) {
	for k, v := range c.nodeGroups {
		ngrf, err := v.GroupReplicationFactor(ctx, a)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", k, err)
		}

		grf += ngrf
	}

	return
}

// Name returns name of the cluster
func (c *Cluster) Name() string {
	return c.name
}

// NodeGroups returns map of node groups in the cluster
func (c *Cluster) NodeGroups() (l map[string]*NodeGroup) {
	return c.nodeGroups
}

// NodeGroupsSorted returns sorted list of node names in the node group
func (c *Cluster) NodeGroupsSorted() (l []string) {
	l = make([]string, len(c.nodeGroups))

	i := 0
	for k := range c.nodeGroups {
		l[i] = k
		i++
	}
	sort.Strings(l)

	return
}

// NodeGroup returns node group
func (c *Cluster) NodeGroup(name string) *NodeGroup {
	return c.nodeGroups[name]
}

// Nodes returns map of nodes in the cluster
func (c *Cluster) Nodes() map[string]*Node {
	n := make(map[string]*Node)
	for _, ng := range c.NodeGroups() {
		for k, v := range ng.getNodes() {
			n[k] = v
		}
	}
	return n
}

// NodesClients returns map of node's clients in the cluster excluding stopped nodes
func (c *Cluster) NodesClients(ctx context.Context) (map[string]*Client, error) {
	clients := make(map[string]*Client)
	for _, ng := range c.NodeGroups() {
		ngc, err := ng.NodesClients(ctx)
		if err != nil {
			return nil, fmt.Errorf("nodes clients: %w", err)
		}
		for n, client := range ngc {
			clients[n] = client
		}
	}
	return clients, nil
}

// NodesClientsAll returns map of node's clients in the cluster
func (c *Cluster) NodesClientsAll(ctx context.Context) (map[string]*Client, error) {
	clients := make(map[string]*Client)
	for _, ng := range c.NodeGroups() {
		for n, client := range ng.NodesClientsAll(ctx) {
			clients[n] = client
		}
	}
	return clients, nil
}

// ClusterOverlays represents overlay addresses of all nodes in the cluster
type ClusterOverlays map[string]NodeGroupOverlays

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

// Overlays returns ClusterOverlays
func (c *Cluster) Overlays(ctx context.Context) (overlays ClusterOverlays, err error) {
	overlays = make(ClusterOverlays)

	for k, v := range c.nodeGroups {
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
func (c *Cluster) Peers(ctx context.Context) (peers ClusterPeers, err error) {
	peers = make(ClusterPeers)

	for k, v := range c.nodeGroups {
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
func (c *Cluster) Settlements(ctx context.Context) (settlements ClusterSettlements, err error) {
	settlements = make(ClusterSettlements)

	for k, v := range c.nodeGroups {
		s, err := v.Settlements(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		settlements[k] = s
	}

	return
}

// Size returns size of the cluster
func (c *Cluster) Size() (size int) {
	for _, ng := range c.nodeGroups {
		size += len(ng.nodes)
	}
	return
}

// ClusterTopologies represents Kademlia topology of all nodes in the cluster
type ClusterTopologies map[string]NodeGroupTopologies

// Topologies returns ClusterTopologies
func (c *Cluster) Topologies(ctx context.Context) (topologies ClusterTopologies, err error) {
	topologies = make(ClusterTopologies)

	for k, v := range c.nodeGroups {
		t, err := v.Topologies(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		topologies[k] = t
	}

	return
}

// apiURL generates URL for node's API
func (c *Cluster) apiURL(name string) (u *url.URL, err error) {
	if c.disableNamespace {
		u, err = url.Parse(fmt.Sprintf("%s://%s.%s", c.apiScheme, name, c.apiDomain))
	} else {
		u, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", c.apiScheme, name, c.namespace, c.apiDomain))
	}
	if err != nil {
		return nil, fmt.Errorf("bad API url for node %s: %w", name, err)
	}
	return
}

// ingressHost generates host for node's API ingress
func (c *Cluster) ingressHost(name string) string {
	if c.disableNamespace {
		return fmt.Sprintf("%s.%s", name, c.apiDomain)
	}
	return fmt.Sprintf("%s.%s.%s", name, c.namespace, c.apiDomain)
}

// debugAPIURL generates URL for node's DebugAPI
func (c *Cluster) debugAPIURL(name string) (u *url.URL, err error) {
	if c.disableNamespace {
		u, err = url.Parse(fmt.Sprintf("%s://%s-debug.%s", c.debugAPIScheme, name, c.debugAPIDomain))
	} else {
		u, err = url.Parse(fmt.Sprintf("%s://%s-debug.%s.%s", c.debugAPIScheme, name, c.namespace, c.debugAPIDomain))
	}
	if err != nil {
		return nil, fmt.Errorf("bad debug API url for node %s: %w", name, err)
	}
	return
}

// ingressHost generates host for node's DebugAPI ingress
func (c *Cluster) ingressDebugHost(name string) string {
	if c.disableNamespace {
		return fmt.Sprintf("%s-debug.%s", name, c.debugAPIDomain)
	}
	return fmt.Sprintf("%s-debug.%s.%s", name, c.namespace, c.debugAPIDomain)
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

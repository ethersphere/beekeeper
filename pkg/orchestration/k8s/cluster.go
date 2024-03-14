package k8s

import (
	"context"
	"fmt"
	"math/rand"
	"sort"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// compile check whether client implements interface
var _ orchestration.Cluster = (*Cluster)(nil)

// Cluster represents cluster of Bee nodes
type Cluster struct {
	name       string
	opts       orchestration.ClusterOptions
	NodeGroups map[string]orchestration.NodeGroup // set when groups are added to the cluster
	logger     logging.Logger
}

// NewCluster returns new cluster
func NewCluster(name string, o orchestration.ClusterOptions, logger logging.Logger) *Cluster {
	return &Cluster{
		name:       name,
		opts:       o,
		NodeGroups: make(map[string]orchestration.NodeGroup),
		logger:     logger,
	}
}

// AddNodeGroup adds new node group to the cluster
func (c *Cluster) AddNodeGroup(name string, o orchestration.NodeGroupOptions) {
	o.Annotations = mergeMaps(c.opts.Annotations, o.Annotations)
	o.Labels = mergeMaps(c.opts.Labels, o.Labels)
	g := NewNodeGroup(name, o, c.opts, c.logger)
	g.k8s = c.opts.K8SClient
	c.NodeGroups[name] = g
}

// Addresses returns ClusterAddresses
func (c *Cluster) Addresses(ctx context.Context) (addrs map[string]orchestration.NodeGroupAddresses, err error) {
	addrs = make(orchestration.ClusterAddresses)

	for k, v := range c.NodeGroups {
		a, err := v.Addresses(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		addrs[k] = a
	}

	return
}

// Accounting returns ClusterAccounting
func (c *Cluster) Accounting(ctx context.Context) (accounting orchestration.ClusterAccounting, err error) {
	accounting = make(orchestration.ClusterAccounting)

	for k, v := range c.NodeGroups {
		a, err := v.Accounting(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		accounting[k] = a
	}

	return
}

// FlattenBalances returns aggregated NodeGroupBalances
func (c *Cluster) FlattenAccounting(ctx context.Context) (accounting orchestration.NodeGroupAccounting, err error) {
	a, err := c.Accounting(ctx)
	if err != nil {
		return nil, err
	}

	accounting = make(orchestration.NodeGroupAccounting)

	for _, v := range a {
		for n, acc := range v {
			if _, found := accounting[n]; found {
				return nil, fmt.Errorf("key %s already present", n)
			}
			accounting[n] = acc
		}
	}

	return
}

// Balances returns ClusterBalances
func (c *Cluster) Balances(ctx context.Context) (balances orchestration.ClusterBalances, err error) {
	balances = make(orchestration.ClusterBalances)

	for k, v := range c.NodeGroups {
		b, err := v.Balances(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		balances[k] = b
	}

	return
}

// FlattenBalances returns aggregated NodeGroupBalances
func (c *Cluster) FlattenBalances(ctx context.Context) (balances orchestration.NodeGroupBalances, err error) {
	b, err := c.Balances(ctx)
	if err != nil {
		return nil, err
	}

	balances = make(orchestration.NodeGroupBalances)

	for _, v := range b {
		for n, bal := range v {
			if _, found := balances[n]; found {
				return nil, fmt.Errorf("key %s already present", n)
			}
			balances[n] = bal
		}
	}

	return
}

// GlobalReplicationFactor returns the total number of nodes in the cluster that contain given chunk
func (c *Cluster) GlobalReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error) {
	for k, v := range c.NodeGroups {
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

// NodeGroupsMap returns map of node groups in the cluster
func (c *Cluster) NodeGroupsMap() (l map[string]orchestration.NodeGroup) {
	nodeGroups := make(map[string]orchestration.NodeGroup)
	for k, v := range c.NodeGroups {
		nodeGroups[k] = v
	}
	return nodeGroups
}

// NodeGroupsSorted returns sorted list of node names in the node group
func (c *Cluster) NodeGroupsSorted() (l []string) {
	l = make([]string, len(c.NodeGroups))

	i := 0
	for k := range c.NodeGroups {
		l[i] = k
		i++
	}
	sort.Strings(l)

	return
}

// NodeGroup returns node group
func (c *Cluster) NodeGroup(name string) (ng orchestration.NodeGroup, err error) {
	ng, ok := c.NodeGroups[name]
	if !ok {
		return nil, fmt.Errorf("node group %s not found", name)
	}
	return
}

// Nodes returns map of nodes in the cluster
func (c *Cluster) Nodes() map[string]orchestration.Node {
	n := make(map[string]orchestration.Node)
	for _, ng := range c.NodeGroupsMap() {
		for k, v := range ng.NodesMap() {
			n[k] = v
		}
	}
	return n
}

// NodeNamess returns a list of node names in the cluster across all node groups
func (c *Cluster) NodeNames() (names []string) {
	for _, ng := range c.NodeGroupsMap() {
		for k := range ng.NodesMap() {
			names = append(names, k)
		}
	}

	return
}

// LightNodeNames returns a list of light node names
func (c *Cluster) LightNodeNames() (names []string) {
	for name, node := range c.Nodes() {
		if !node.Config().FullNode {
			names = append(names, name)
		}
	}
	return
}

// FullNodeNames returns a list of full node names
func (c *Cluster) FullNodeNames() (names []string) {
	for name, node := range c.Nodes() {
		cfg := node.Config()
		if cfg.FullNode && !cfg.BootnodeMode {
			names = append(names, name)
		}
	}
	return
}

// NodesClients returns map of node's clients in the cluster excluding stopped nodes
func (c *Cluster) NodesClients(ctx context.Context) (map[string]*bee.Client, error) {
	clients := make(map[string]*bee.Client)
	for _, ng := range c.NodeGroupsMap() {
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
func (c *Cluster) NodesClientsAll(ctx context.Context) (map[string]*bee.Client, error) {
	clients := make(map[string]*bee.Client)
	for _, ng := range c.NodeGroupsMap() {
		for n, client := range ng.NodesClientsAll(ctx) {
			clients[n] = client
		}
	}
	return clients, nil
}

// Overlays returns ClusterOverlays excluding the provided node group names
func (c *Cluster) Overlays(ctx context.Context, exclude ...string) (overlays orchestration.ClusterOverlays, err error) {
	overlays = make(orchestration.ClusterOverlays)

	for k, v := range c.NodeGroups {
		if containsName(exclude, k) {
			continue
		}
		o, err := v.Overlays(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		overlays[k] = o
	}

	return
}

// FlattenOverlays returns aggregated ClusterOverlays excluding the provided node group names
func (c *Cluster) FlattenOverlays(ctx context.Context, exclude ...string) (map[string]swarm.Address, error) {
	o, err := c.Overlays(ctx)
	if err != nil {
		return nil, err
	}

	res := make(map[string]swarm.Address)

	for ngn, ngo := range o {
		if containsName(exclude, ngn) {
			continue
		}
		for n, over := range ngo {
			if _, found := res[n]; found {
				return nil, fmt.Errorf("key %s already present", n)
			}
			res[n] = over
		}
	}

	return res, nil
}

func containsName(s []string, e string) bool {
	for i := range s {
		if s[i] == e {
			return true
		}
	}
	return false
}

// Peers returns peers of all nodes in the cluster
func (c *Cluster) Peers(ctx context.Context, exclude ...string) (peers orchestration.ClusterPeers, err error) {
	peers = make(orchestration.ClusterPeers)

	for k, v := range c.NodeGroups {
		if containsName(exclude, k) {
			continue
		}
		p, err := v.Peers(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		peers[k] = p
	}

	return
}

// RandomNode returns random running node from a cluster
func (c *Cluster) RandomNode(ctx context.Context, r *rand.Rand) (node orchestration.Node, err error) {
	nodes := []orchestration.Node{}
	for _, ng := range c.NodeGroupsMap() {
		stopped, err := ng.StoppedNodes(ctx)
		if err != nil && err != orchestration.ErrNotSet {
			return nil, fmt.Errorf("stopped nodes: %w", err)
		}

		for _, v := range ng.NodesMap() {
			if contains(stopped, v.Name()) {
				continue
			}
			nodes = append(nodes, v)
		}
	}

	return nodes[r.Intn(len(nodes))], nil
}

// Settlements returns
func (c *Cluster) Settlements(ctx context.Context) (settlements orchestration.ClusterSettlements, err error) {
	settlements = make(orchestration.ClusterSettlements)

	for k, v := range c.NodeGroups {
		s, err := v.Settlements(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		settlements[k] = s
	}

	return
}

// FlattenSettlements returns aggregated NodeGroupSettlements
func (c *Cluster) FlattenSettlements(ctx context.Context) (settlements orchestration.NodeGroupSettlements, err error) {
	s, err := c.Settlements(ctx)
	if err != nil {
		return nil, err
	}

	settlements = make(orchestration.NodeGroupSettlements)

	for _, v := range s {
		for n, set := range v {
			if _, found := settlements[n]; found {
				return nil, fmt.Errorf("key %s already present", n)
			}
			settlements[n] = set
		}
	}

	return
}

// Size returns size of the cluster
func (c *Cluster) Size() (size int) {
	for _, ng := range c.NodeGroups {
		size += ng.Size()
	}
	return
}

// Topologies returns ClusterTopologies
func (c *Cluster) Topologies(ctx context.Context) (topologies orchestration.ClusterTopologies, err error) {
	topologies = make(orchestration.ClusterTopologies)

	for k, v := range c.NodeGroups {
		t, err := v.Topologies(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}

		topologies[k] = t
	}

	return
}

// FlattenTopologies returns an aggregate of Topologies
func (c *Cluster) FlattenTopologies(ctx context.Context) (topologies map[string]bee.Topology, err error) {
	top, err := c.Topologies(ctx)
	if err != nil {
		return nil, err
	}

	topologies = make(map[string]bee.Topology)

	for _, v := range top {
		for n, over := range v {
			if _, found := topologies[n]; found {
				return nil, fmt.Errorf("key %s already present", n)
			}
			topologies[n] = over
		}
	}

	return
}

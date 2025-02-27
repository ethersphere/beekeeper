package k8s

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/httpx"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/orchestration/notset"
	"github.com/ethersphere/beekeeper/pkg/swap"
)

// compile check whether client implements interface
var _ orchestration.Cluster = (*Cluster)(nil)

// Cluster represents cluster of Bee nodes
type Cluster struct {
	nodeOrchestrator orchestration.NodeOrchestrator
	name             string
	opts             orchestration.ClusterOptions
	nodeGroups       map[string]orchestration.NodeGroup // set when groups are added to the cluster
	httpClient       *http.Client
	k8sClient        *k8s.Client
	swapClient       swap.Client
	log              logging.Logger
}

// NewCluster returns new cluster
func NewCluster(name string, o orchestration.ClusterOptions, k8s *k8s.Client, swapClient swap.Client, log logging.Logger) *Cluster {
	var nodeOrchestrator orchestration.NodeOrchestrator

	if k8s == nil {
		nodeOrchestrator = &notset.BeeClient{}
	} else {
		nodeOrchestrator = newNodeOrchestrator(k8s, log)
	}

	if swapClient == nil {
		swapClient = &swap.NotSet{}
	}

	return &Cluster{
		name:             name,
		nodeOrchestrator: nodeOrchestrator,
		opts:             o,
		nodeGroups:       make(map[string]orchestration.NodeGroup),
		log:              log,
		httpClient: &http.Client{
			Transport: &httpx.HeaderRoundTripper{
				Next: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: o.APIInsecureTLS,
					},
				},
			},
		},
		k8sClient:  k8s,
		swapClient: swapClient,
	}
}

// AddNodeGroup adds new node group to the cluster
func (c *Cluster) AddNodeGroup(name string, o orchestration.NodeGroupOptions) {
	c.nodeGroups[name] = NewNodeGroup(name, c.opts, c.nodeOrchestrator, o, c.httpClient, c.swapClient, c.k8sClient, c.log)
}

// Addresses returns ClusterAddresses
func (c *Cluster) Addresses(ctx context.Context) (addrs map[string]orchestration.NodeGroupAddresses, err error) {
	addrs = make(orchestration.ClusterAddresses)

	for k, v := range c.nodeGroups {
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

	for k, v := range c.nodeGroups {
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

	for k, v := range c.nodeGroups {
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
func (c *Cluster) NodeGroups() (l map[string]orchestration.NodeGroup) {
	nodeGroups := make(map[string]orchestration.NodeGroup)
	for k, v := range c.nodeGroups {
		nodeGroups[k] = v
	}
	return nodeGroups
}

// NodeGroup returns node group
func (c *Cluster) NodeGroup(name string) (ng orchestration.NodeGroup, err error) {
	ng, ok := c.nodeGroups[name]
	if !ok {
		return nil, fmt.Errorf("node group %s not found", name)
	}
	return
}

// Nodes returns map of nodes in the cluster
func (c *Cluster) Nodes() map[string]orchestration.Node {
	n := make(map[string]orchestration.Node)
	for _, ng := range c.NodeGroups() {
		for k, v := range ng.Nodes() {
			n[k] = v
		}
	}
	return n
}

// NodeNamess returns a list of node names in the cluster across all node groups
func (c *Cluster) NodeNames() (names []string) {
	for _, ng := range c.NodeGroups() {
		for k := range ng.Nodes() {
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

// ShuffledFullNodeClients returns a shuffled list of full node clients
func (c *Cluster) ShuffledFullNodeClients(ctx context.Context, r *rand.Rand) ([]*bee.Client, error) {
	var res []*bee.Client
	for _, node := range c.Nodes() {
		cfg := node.Config()
		if cfg.FullNode && !cfg.BootnodeMode {
			res = append(res, node.Client())
		}
	}
	r.Shuffle(len(res), func(i, j int) {
		res[i], res[j] = res[j], res[i]
	})
	return res, nil
}

// NodesClients returns map of node's clients in the cluster excluding stopped nodes
func (c *Cluster) NodesClients(ctx context.Context) (map[string]*bee.Client, error) {
	clients := make(map[string]*bee.Client)
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

// Overlays returns ClusterOverlays excluding the provided node group names
func (c *Cluster) Overlays(ctx context.Context, exclude ...string) (overlays orchestration.ClusterOverlays, err error) {
	overlays = make(orchestration.ClusterOverlays)

	for k, v := range c.nodeGroups {
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

	for k, v := range c.nodeGroups {
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
	for _, ng := range c.NodeGroups() {
		stopped, err := ng.StoppedNodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("stopped nodes: %w", err)
		}

		for _, v := range ng.Nodes() {
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

	for k, v := range c.nodeGroups {
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
	for _, ng := range c.nodeGroups {
		size += ng.Size()
	}
	return
}

// Topologies returns ClusterTopologies
func (c *Cluster) Topologies(ctx context.Context) (topologies orchestration.ClusterTopologies, err error) {
	topologies = make(orchestration.ClusterTopologies)

	for k, v := range c.nodeGroups {
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

// ClosetFullNodeClient returns the closest full node client to the supplied node.
func (c *Cluster) ClosetFullNodeClient(ctx context.Context, s *bee.Client, r *rand.Rand) (*bee.Client, error) {
	addrToNode := make(map[string]orchestration.Node)
	for _, n := range c.Nodes() {
		res, err := n.Client().Addresses(ctx)
		if err != nil {
			return nil, err
		}
		addrToNode[res.Overlay.String()] = n
	}

	t, err := s.Topology(ctx)
	if err != nil {
		return nil, err
	}
	const maxBin = 32
	for b := range maxBin {
		bin := t.Bins[fmt.Sprintf("bin_%d", b)]
		var fullNodes []orchestration.Node
		for _, peer := range bin.ConnectedPeers {
			node, ok := addrToNode[peer.Address.String()]
			if !ok {
				return nil, fmt.Errorf("peer overlay %s not found in address map", peer.Address.String())
			}
			cfg := node.Config()
			if cfg.FullNode && !cfg.BootnodeMode {
				fullNodes = append(fullNodes, node)
			}
		}
		if len(fullNodes) == 0 {
			continue
		}
		r.Shuffle(len(fullNodes), func(i, j int) {
			fullNodes[i], fullNodes[j] = fullNodes[j], fullNodes[i]
		})
		return fullNodes[0].Client(), nil
	}
	return nil, fmt.Errorf("cannot find closest fullnode")
}

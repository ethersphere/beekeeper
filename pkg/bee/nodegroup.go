package bee

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/ethersphere/bee/pkg/swarm"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

// NodeGroup represents group of Bee nodes
type NodeGroup struct {
	name  string
	nodes map[string]*Client
	opts  NodeGroupOptions
	// set when added to the cluster
	cluster *DynamicCluster
	k8s     *k8sBee.Client
}

// NodeGroupOptions represents node group options
type NodeGroupOptions struct {
	Annotations               map[string]string
	ClefImage                 string
	ClefImagePullPolicy       string
	Image                     string
	ImagePullPolicy           string
	IngressAnnotations        map[string]string
	IngressDebugAnnotations   map[string]string
	Labels                    map[string]string
	LimitCPU                  string
	LimitMemory               string
	NodeSelector              map[string]string
	PersistenceEnabled        bool
	PersistenceStorageClass   string
	PersistanceStorageRequest string
	PodManagementPolicy       string
	RestartPolicy             string
	RequestCPU                string
	RequestMemory             string
	UpdateStrategy            string
}

// NewNodeGroup returns new node group
func NewNodeGroup(name string, o NodeGroupOptions) *NodeGroup {
	return &NodeGroup{
		name:  name,
		nodes: make(map[string]*Client),
		opts:  o,
	}
}

// AddNode adss new node to the node group
func (g *NodeGroup) AddNode(name string) (err error) {
	aURL, err := g.cluster.apiURL(name)
	if err != nil {
		return fmt.Errorf("adding node %s: %s", name, err)
	}

	dURL, err := g.cluster.debugAPIURL(name)
	if err != nil {
		return fmt.Errorf("adding node %s: %s", name, err)
	}

	c := NewClient(ClientOptions{
		APIURL:              aURL,
		APIInsecureTLS:      g.cluster.apiInsecureTLS,
		DebugAPIURL:         dURL,
		DebugAPIInsecureTLS: g.cluster.debugAPIInsecureTLS,
	})
	g.nodes[name] = &c

	return
}

// NodeGroupAddresses represents addresses of all nodes in the node group
type NodeGroupAddresses map[string]Addresses

// Addresses returns NodeGroupAddresses
func (g *NodeGroup) Addresses(ctx context.Context) (addrs NodeGroupAddresses, err error) {
	addrs = make(NodeGroupAddresses)

	var msgs []AddressesStreamMsg2
	for m := range g.AddressesStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
		}
		addrs[m.Name] = m.Addresses
	}

	return
}

// AddressesStreamMsg2 represents message sent over the AddressStream channel
type AddressesStreamMsg2 struct {
	Name      string
	Addresses Addresses
	Error     error
}

// AddressesStream returns stream of addresses of all nodes in the node group
func (g *NodeGroup) AddressesStream(ctx context.Context) <-chan AddressesStreamMsg2 {
	addressStream := make(chan AddressesStreamMsg2)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Addresses(ctx)
			addressStream <- AddressesStreamMsg2{
				Name:      n,
				Addresses: a,
				Error:     err,
			}
		}(k, v)
	}

	go func() {
		wg.Wait()
		close(addressStream)
	}()

	return addressStream
}

// NodeGroupBalances represents balances of all nodes in the node group
type NodeGroupBalances map[string]Balances

// Balances returns NodeGroupBalances
func (g *NodeGroup) Balances(ctx context.Context) (balances NodeGroupBalances, err error) {
	balances = make(NodeGroupBalances)

	var msgs []BalancesStreamMsg2
	for m := range g.BalancesStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
		}

		balances[m.Name] = m.Balances
	}

	return
}

// BalancesStreamMsg2 represents message sent over the BalancesStream channel
type BalancesStreamMsg2 struct {
	Name     string
	Balances Balances
	Error    error
}

// BalancesStream returns stream of balances of all nodes in the cluster
func (g *NodeGroup) BalancesStream(ctx context.Context) <-chan BalancesStreamMsg2 {
	balancesStream := make(chan BalancesStreamMsg2)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			b, err := c.Balances(ctx)
			balancesStream <- BalancesStreamMsg2{
				Name:     n,
				Balances: b,
				Error:    err,
			}
		}(k, v)
	}

	go func() {
		wg.Wait()
		close(balancesStream)
	}()

	return balancesStream
}

// GroupReplicationFactor returns the total number of nodes in the node group that contain given chunk
func (g *NodeGroup) GroupReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error) {
	var msgs []HasChunkStreamMsg2
	for m := range g.HasChunkStream(ctx, a) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return 0, fmt.Errorf("%s: %w", m.Name, m.Error)
		}
		if m.Found {
			grf++
		}
	}

	return
}

// HasChunkStreamMsg2 represents message sent over the HasChunkStream channel
type HasChunkStreamMsg2 struct {
	Name  string
	Found bool
	Error error
}

// HasChunkStream returns stream of HasChunk requests for all nodes in the node group
func (g *NodeGroup) HasChunkStream(ctx context.Context, a swarm.Address) <-chan HasChunkStreamMsg2 {
	hasChunkStream := make(chan HasChunkStreamMsg2)

	go func() {
		var wg sync.WaitGroup
		for k, v := range g.nodes {
			wg.Add(1)
			go func(n string, c *Client) {
				defer wg.Done()

				found, err := c.HasChunk(ctx, a)
				hasChunkStream <- HasChunkStreamMsg2{
					Name:  n,
					Found: found,
					Error: err,
				}
			}(k, v)
		}

		wg.Wait()
		close(hasChunkStream)
	}()

	return hasChunkStream
}

// Name returns name of the node group
func (g *NodeGroup) Name() string {
	return g.name
}

// Nodes returns map of node groups in the node group
func (g *NodeGroup) Nodes() (l map[string]*Client) {
	return g.nodes
}

// NodesSorted returns sorted list of node names in the node group
func (g *NodeGroup) NodesSorted() (l []string) {
	l = make([]string, len(g.nodes))

	i := 0
	for k := range g.nodes {
		l[i] = k
		i++
	}
	sort.Strings(l)

	return
}

// Node returns node's client
func (g *NodeGroup) Node(name string) *Client {
	return g.nodes[name]
}

// NodeGroupOverlays represents overlay addresses of all nodes in the node group
type NodeGroupOverlays map[string]swarm.Address

// Overlays returns NodeGroupOverlays
func (g *NodeGroup) Overlays(ctx context.Context) (overlays NodeGroupOverlays, err error) {
	overlays = make(NodeGroupOverlays)

	var msgs []OverlaysStreamMsg2
	for m := range g.OverlaysStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, m.Error
		}
		overlays[m.Name] = m.Address
	}

	return
}

// OverlaysStreamMsg2 represents message sent over the OverlaysStream channel
type OverlaysStreamMsg2 struct {
	Name    string
	Address swarm.Address
	Error   error
}

// OverlaysStream returns stream of overlay addresses of all nodes in the node group
// TODO: add semaphore
func (g *NodeGroup) OverlaysStream(ctx context.Context) <-chan OverlaysStreamMsg2 {
	overlaysStream := make(chan OverlaysStreamMsg2)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Overlay(ctx)
			overlaysStream <- OverlaysStreamMsg2{
				Name:    n,
				Address: a,
				Error:   err,
			}
		}(k, v)
	}

	go func() {
		wg.Wait()
		close(overlaysStream)
	}()

	return overlaysStream
}

// NodeGroupPeers represents peers of all nodes in the node group
type NodeGroupPeers map[string][]swarm.Address

// Peers returns NodeGroupPeers
func (g *NodeGroup) Peers(ctx context.Context) (peers NodeGroupPeers, err error) {
	peers = make(NodeGroupPeers)

	var msgs []PeersStreamMsg2
	for m := range g.PeersStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
		}
		peers[m.Name] = m.Peers
	}

	return
}

// PeersStreamMsg2 represents message sent over the PeersStream channel
type PeersStreamMsg2 struct {
	Name  string
	Peers []swarm.Address
	Error error
}

// PeersStream returns stream of peers of all nodes in the node group
func (g *NodeGroup) PeersStream(ctx context.Context) <-chan PeersStreamMsg2 {
	peersStream := make(chan PeersStreamMsg2)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Peers(ctx)
			peersStream <- PeersStreamMsg2{
				Name:  n,
				Peers: a,
				Error: err,
			}
		}(k, v)
	}

	go func() {
		wg.Wait()
		close(peersStream)
	}()

	return peersStream
}

// RemoveNode removes node from the node group
func (g *NodeGroup) RemoveNode(name string) {
	delete(g.nodes, name)
}

// NodeGroupSettlements represents settlements of all nodes in the node group
type NodeGroupSettlements map[string]map[string]SentReceived

// SentReceived object
type SentReceived struct {
	Received int
	Sent     int
}

// Settlements returns NodeGroupSettlements
func (g *NodeGroup) Settlements(ctx context.Context) (settlements NodeGroupSettlements, err error) {
	settlements = make(NodeGroupSettlements)

	var msgs []SettlementsStreamMsg2
	for m := range g.SettlementsStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
		}

		tmp := make(map[string]SentReceived)
		for _, s := range m.Settlements.Settlements {
			tmp[s.Peer] = SentReceived{
				Received: s.Received,
				Sent:     s.Sent,
			}
		}
		settlements[m.Name] = tmp
	}

	return
}

// SettlementsStreamMsg2 represents message sent over the SettlementsStream channel
type SettlementsStreamMsg2 struct {
	Name        string
	Settlements Settlements
	Error       error
}

// SettlementsStream returns stream of settlements of all nodes in the cluster
func (g *NodeGroup) SettlementsStream(ctx context.Context) <-chan SettlementsStreamMsg2 {
	SettlementsStream := make(chan SettlementsStreamMsg2)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			s, err := c.Settlements(ctx)
			SettlementsStream <- SettlementsStreamMsg2{
				Name:        n,
				Settlements: s,
				Error:       err,
			}
		}(k, v)
	}

	go func() {
		wg.Wait()
		close(SettlementsStream)
	}()

	return SettlementsStream
}

// Size returns size of the node group
func (g *NodeGroup) Size() int {
	return len(g.nodes)
}

// StartNodeOptions represents node start options
type StartNodeOptions struct {
	Name         string
	Config       k8sBee.Config
	ClefKey      string
	ClefPassword string
	LibP2PKey    string
	SwarmKey     string
}

// StartNode starts new node in the node group
func (g *NodeGroup) StartNode(ctx context.Context, o StartNodeOptions) (err error) {
	if err := g.AddNode(o.Name); err != nil {
		return fmt.Errorf("starting node %s: %s", o.Name, err)
	}

	labels := mergeMaps(g.opts.Labels, map[string]string{
		"app.kubernetes.io/instance": o.Name,
	})

	if err := g.k8s.NodeStart(ctx, k8sBee.NodeStartOptions{
		// Bee configuration
		Config: o.Config,
		// Kubernetes configuration
		Name:                      o.Name,
		Namespace:                 g.cluster.namespace,
		Annotations:               g.opts.Annotations,
		ClefImage:                 g.opts.ClefImage,
		ClefImagePullPolicy:       g.opts.ClefImagePullPolicy,
		ClefKey:                   o.ClefKey,
		ClefPassword:              o.ClefPassword,
		Image:                     g.opts.Image,
		ImagePullPolicy:           g.opts.ImagePullPolicy,
		IngressAnnotations:        g.opts.IngressAnnotations,
		IngressHost:               g.cluster.ingressHost(o.Name),
		IngressDebugAnnotations:   g.opts.IngressDebugAnnotations,
		IngressDebugHost:          g.cluster.ingressDebugHost(o.Name),
		Labels:                    labels,
		LibP2PKey:                 o.LibP2PKey,
		LimitCPU:                  g.opts.LimitCPU,
		LimitMemory:               g.opts.LimitMemory,
		NodeSelector:              g.opts.NodeSelector,
		PersistenceEnabled:        g.opts.PersistenceEnabled,
		PersistenceStorageClass:   g.opts.PersistenceStorageClass,
		PersistanceStorageRequest: g.opts.PersistanceStorageRequest,
		PodManagementPolicy:       g.opts.PodManagementPolicy,
		RestartPolicy:             g.opts.RestartPolicy,
		RequestCPU:                g.opts.RequestCPU,
		RequestMemory:             g.opts.RequestMemory,
		Selector:                  labels,
		SwarmKey:                  o.SwarmKey,
		UpdateStrategy:            g.opts.UpdateStrategy,
	}); err != nil {
		return fmt.Errorf("starting node %s: %s", o.Name, err)
	}

	return
}

// NodeGroupTopologies represents Kademlia topology of all nodes in the node group
type NodeGroupTopologies map[string]Topology

// Topologies returns NodeGroupTopologies
func (g *NodeGroup) Topologies(ctx context.Context) (topologies NodeGroupTopologies, err error) {
	topologies = make(NodeGroupTopologies)

	var msgs []TopologyStreamMsg2
	for m := range g.TopologyStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
		}
		topologies[m.Name] = m.Topology
	}

	return
}

// TopologyStreamMsg2 represents message sent over the TopologyStream channel
type TopologyStreamMsg2 struct {
	Name     string
	Topology Topology
	Error    error
}

// TopologyStream returns stream of peers of all nodes in the node group
func (g *NodeGroup) TopologyStream(ctx context.Context) <-chan TopologyStreamMsg2 {
	topologyStream := make(chan TopologyStreamMsg2)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			t, err := c.Topology(ctx)
			topologyStream <- TopologyStreamMsg2{
				Name:     n,
				Topology: t,
				Error:    err,
			}
		}(k, v)
	}

	go func() {
		wg.Wait()
		close(topologyStream)
	}()

	return topologyStream
}

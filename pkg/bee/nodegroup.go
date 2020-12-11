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
	cluster *Cluster
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
		return fmt.Errorf("adding node %s: %v", name, err)
	}

	dURL, err := g.cluster.debugAPIURL(name)
	if err != nil {
		return fmt.Errorf("adding node %s: %v", name, err)
	}

	c := NewClient(ClientOptions{
		APIURL:              aURL,
		APIInsecureTLS:      g.cluster.apiInsecureTLS,
		DebugAPIURL:         dURL,
		DebugAPIInsecureTLS: g.cluster.debugAPIInsecureTLS,
	})
	g.nodes[name] = c

	return
}

// NodeGroupAddresses represents addresses of all nodes in the node group
type NodeGroupAddresses map[string]Addresses

// Addresses returns NodeGroupAddresses
func (g *NodeGroup) Addresses(ctx context.Context) (addrs NodeGroupAddresses, err error) {
	addrs = make(NodeGroupAddresses)

	var msgs []AddressesStreamMsg
	for m := range g.AddressesStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %v", m.Name, m.Error)
		}
		addrs[m.Name] = m.Addresses
	}

	return
}

// AddressesStreamMsg represents message sent over the AddressStream channel
type AddressesStreamMsg struct {
	Name      string
	Addresses Addresses
	Error     error
}

// AddressesStream returns stream of addresses of all nodes in the node group
func (g *NodeGroup) AddressesStream(ctx context.Context) <-chan AddressesStreamMsg {
	addressStream := make(chan AddressesStreamMsg)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Addresses(ctx)
			addressStream <- AddressesStreamMsg{
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
type NodeGroupBalances map[string]map[string]int

// Balances returns NodeGroupBalances
func (g *NodeGroup) Balances(ctx context.Context) (balances NodeGroupBalances, err error) {
	balances = make(NodeGroupBalances)

	overlays, err := g.Overlays(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking balances: %v", err)
	}

	var msgs []BalancesStreamMsg
	for m := range g.BalancesStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %v", m.Name, m.Error)
		}

		tmp := make(map[string]int)
		for _, b := range m.Balances.Balances {
			tmp[b.Peer] = b.Balance
		}
		balances[overlays[m.Name].String()] = tmp
	}

	return
}

// BalancesStreamMsg represents message sent over the BalancesStream channel
type BalancesStreamMsg struct {
	Name     string
	Balances Balances
	Error    error
}

// BalancesStream returns stream of balances of all nodes in the cluster
func (g *NodeGroup) BalancesStream(ctx context.Context) <-chan BalancesStreamMsg {
	balancesStream := make(chan BalancesStreamMsg)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			b, err := c.Balances(ctx)
			balancesStream <- BalancesStreamMsg{
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

// DeleteNode deletes node from the k8s cluster and removes it from the node group
func (g *NodeGroup) DeleteNode(ctx context.Context, name string) (err error) {
	if err := g.k8s.NodeDelete(ctx, k8sBee.NodeDeleteOptions{
		Name:      name,
		Namespace: g.cluster.namespace,
	}); err != nil {
		return fmt.Errorf("deleting node %s: %v", name, err)
	}
	g.RemoveNode(name)

	return
}

// GroupReplicationFactor returns the total number of nodes in the node group that contain given chunk
func (g *NodeGroup) GroupReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error) {
	var msgs []HasChunkStreamMsg
	for m := range g.HasChunkStream(ctx, a) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return 0, fmt.Errorf("%s: %v", m.Name, m.Error)
		}
		if m.Found {
			grf++
		}
	}

	return
}

// HasChunkStreamMsg represents message sent over the HasChunkStream channel
type HasChunkStreamMsg struct {
	Name  string
	Found bool
	Error error
}

// HasChunkStream returns stream of HasChunk requests for all nodes in the node group
func (g *NodeGroup) HasChunkStream(ctx context.Context, a swarm.Address) <-chan HasChunkStreamMsg {
	hasChunkStream := make(chan HasChunkStreamMsg)

	go func() {
		var wg sync.WaitGroup
		for k, v := range g.nodes {
			wg.Add(1)
			go func(n string, c *Client) {
				defer wg.Done()

				found, err := c.HasChunk(ctx, a)
				hasChunkStream <- HasChunkStreamMsg{
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

	var msgs []OverlaysStreamMsg
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

// OverlaysStreamMsg represents message sent over the OverlaysStream channel
type OverlaysStreamMsg struct {
	Name    string
	Address swarm.Address
	Error   error
}

// OverlaysStream returns stream of overlay addresses of all nodes in the node group
// TODO: add semaphore
func (g *NodeGroup) OverlaysStream(ctx context.Context) <-chan OverlaysStreamMsg {
	overlaysStream := make(chan OverlaysStreamMsg)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Overlay(ctx)
			overlaysStream <- OverlaysStreamMsg{
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

	var msgs []PeersStreamMsg
	for m := range g.PeersStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %v", m.Name, m.Error)
		}
		peers[m.Name] = m.Peers
	}

	return
}

// PeersStreamMsg represents message sent over the PeersStream channel
type PeersStreamMsg struct {
	Name  string
	Peers []swarm.Address
	Error error
}

// PeersStream returns stream of peers of all nodes in the node group
func (g *NodeGroup) PeersStream(ctx context.Context) <-chan PeersStreamMsg {
	peersStream := make(chan PeersStreamMsg)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Peers(ctx)
			peersStream <- PeersStreamMsg{
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

	overlays, err := g.Overlays(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking settlements: %v", err)
	}

	var msgs []SettlementsStreamMsg
	for m := range g.SettlementsStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %v", m.Name, m.Error)
		}

		tmp := make(map[string]SentReceived)
		for _, s := range m.Settlements.Settlements {
			tmp[s.Peer] = SentReceived{
				Received: s.Received,
				Sent:     s.Sent,
			}
		}
		settlements[overlays[m.Name].String()] = tmp
	}

	return
}

// SettlementsStreamMsg represents message sent over the SettlementsStream channel
type SettlementsStreamMsg struct {
	Name        string
	Settlements Settlements
	Error       error
}

// SettlementsStream returns stream of settlements of all nodes in the cluster
func (g *NodeGroup) SettlementsStream(ctx context.Context) <-chan SettlementsStreamMsg {
	SettlementsStream := make(chan SettlementsStreamMsg)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			s, err := c.Settlements(ctx)
			SettlementsStream <- SettlementsStreamMsg{
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
		return fmt.Errorf("starting node %s: %v", o.Name, err)
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
		return fmt.Errorf("starting node %s: %v", o.Name, err)
	}

	return
}

// StopNode stops node by scaling down its statefulset to 0
func (g *NodeGroup) StopNode(ctx context.Context, name string) (err error) {
	if err := g.k8s.NodeStop(ctx, k8sBee.NodeStopOptions{
		Name:      name,
		Namespace: g.cluster.namespace,
	}); err != nil {
		return fmt.Errorf("stopping node %s: %v", name, err)
	}

	return
}

// NodeGroupTopologies represents Kademlia topology of all nodes in the node group
type NodeGroupTopologies map[string]Topology

// Topologies returns NodeGroupTopologies
func (g *NodeGroup) Topologies(ctx context.Context) (topologies NodeGroupTopologies, err error) {
	topologies = make(NodeGroupTopologies)

	var msgs []TopologyStreamMsg
	for m := range g.TopologyStream(ctx) {
		msgs = append(msgs, m)
	}

	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %v", m.Name, m.Error)
		}
		topologies[m.Name] = m.Topology
	}

	return
}

// TopologyStreamMsg represents message sent over the TopologyStream channel
type TopologyStreamMsg struct {
	Name     string
	Topology Topology
	Error    error
}

// TopologyStream returns stream of peers of all nodes in the node group
func (g *NodeGroup) TopologyStream(ctx context.Context) <-chan TopologyStreamMsg {
	topologyStream := make(chan TopologyStreamMsg)

	var wg sync.WaitGroup
	for k, v := range g.nodes {
		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			t, err := c.Topology(ctx)
			topologyStream <- TopologyStreamMsg{
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

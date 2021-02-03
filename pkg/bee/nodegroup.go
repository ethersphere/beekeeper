package bee

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/k8s"
)

const nodeReadyTimeout = 3 * time.Second

// NodeGroup represents group of Bee nodes
type NodeGroup struct {
	name  string
	nodes map[string]*Node
	opts  NodeGroupOptions

	// set when added to the cluster
	cluster *Cluster
	k8s     k8s.Bee

	lock sync.RWMutex
}

// NodeGroupOptions represents node group options
type NodeGroupOptions struct {
	Annotations               map[string]string
	ClefImage                 string
	ClefImagePullPolicy       string
	BeeConfig                 *k8s.Config
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
		nodes: make(map[string]*Node),
		opts:  o,
	}
}

// AddNode adss new node to the node group
func (g *NodeGroup) AddNode(name string, o NodeOptions) (err error) {
	aURL, err := g.cluster.apiURL(name)
	if err != nil {
		return fmt.Errorf("API URL %s: %w", name, err)
	}

	dURL, err := g.cluster.debugAPIURL(name)
	if err != nil {
		return fmt.Errorf("debug API URL %s: %w", name, err)
	}

	client := NewClient(ClientOptions{
		APIURL:              aURL,
		APIInsecureTLS:      g.cluster.apiInsecureTLS,
		DebugAPIURL:         dURL,
		DebugAPIInsecureTLS: g.cluster.debugAPIInsecureTLS,
		Retry:               5,
	})

	// TODO: make more granular, check every sub-option
	var config *k8s.Config
	if o.Config != nil {
		config = o.Config
	} else {
		config = g.opts.BeeConfig
	}

	n := NewNode(name, NodeOptions{
		ClefKey:      o.ClefKey,
		ClefPassword: o.ClefPassword,
		Client:       client,
		Config:       config,
		LibP2PKey:    o.LibP2PKey,
		SwarmKey:     o.SwarmKey,
	})

	g.addNode(n)

	return
}

// AddStartNode adds new node in the node group and starts it in the k8s cluster
func (g *NodeGroup) AddStartNode(ctx context.Context, name string, o NodeOptions) (err error) {
	if err := g.AddNode(name, o); err != nil {
		return fmt.Errorf("add node %s: %w", name, err)
	}

	if err := g.CreateNode(ctx, name); err != nil {
		return fmt.Errorf("create node %s in k8s: %w", name, err)
	}

	if err := g.StartNode(ctx, name); err != nil {
		return fmt.Errorf("start node %s in k8s: %w", name, err)
	}

	return
}

// NodeGroupAddresses represents addresses of all nodes in the node group
type NodeGroupAddresses map[string]Addresses

// Addresses returns NodeGroupAddresses
func (g *NodeGroup) Addresses(ctx context.Context) (addrs NodeGroupAddresses, err error) {
	stream, err := g.AddressesStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("addresses stream: %w", err)
	}

	var msgs []AddressesStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	addrs = make(NodeGroupAddresses)
	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
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
func (g *NodeGroup) AddressesStream(ctx context.Context) (<-chan AddressesStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != k8s.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	addressStream := make(chan AddressesStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.name) {
			continue
		}

		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Addresses(ctx)
			addressStream <- AddressesStreamMsg{
				Name:      n,
				Addresses: a,
				Error:     err,
			}
		}(k, v.client)
	}

	go func() {
		wg.Wait()
		close(addressStream)
	}()

	return addressStream, nil
}

// NodeGroupBalances represents balances of all nodes in the node group
type NodeGroupBalances map[string]map[string]int

// Balances returns NodeGroupBalances
func (g *NodeGroup) Balances(ctx context.Context) (balances NodeGroupBalances, err error) {
	stream, err := g.BalancesStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("balances stream: %w", err)
	}

	overlays, err := g.Overlays(ctx)
	if err != nil {
		return nil, fmt.Errorf("overlays: %w", err)
	}

	var msgs []BalancesStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	balances = make(NodeGroupBalances)
	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
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
func (g *NodeGroup) BalancesStream(ctx context.Context) (<-chan BalancesStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != k8s.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	balancesStream := make(chan BalancesStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.name) {
			continue
		}

		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			b, err := c.Balances(ctx)
			balancesStream <- BalancesStreamMsg{
				Name:     n,
				Balances: b,
				Error:    err,
			}
		}(k, v.client)
	}

	go func() {
		wg.Wait()
		close(balancesStream)
	}()

	return balancesStream, nil
}

// CreateNode creates new node in the k8s cluster
func (g *NodeGroup) CreateNode(ctx context.Context, name string) (err error) {
	labels := mergeMaps(g.opts.Labels, map[string]string{
		"app.kubernetes.io/instance": name,
	})

	n := g.getNode(name)

	if err := g.k8s.Create(ctx, k8s.CreateOptions{
		// Bee configuration
		Config: *n.config,
		// Kubernetes configuration
		Name:                      name,
		Namespace:                 g.cluster.namespace,
		Annotations:               g.opts.Annotations,
		ClefImage:                 g.opts.ClefImage,
		ClefImagePullPolicy:       g.opts.ClefImagePullPolicy,
		ClefKey:                   n.clefKey,
		ClefPassword:              n.clefPassword,
		Image:                     g.opts.Image,
		ImagePullPolicy:           g.opts.ImagePullPolicy,
		IngressAnnotations:        g.opts.IngressAnnotations,
		IngressHost:               g.cluster.ingressHost(name),
		IngressDebugAnnotations:   g.opts.IngressDebugAnnotations,
		IngressDebugHost:          g.cluster.ingressDebugHost(name),
		Labels:                    labels,
		LibP2PKey:                 n.libP2PKey,
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
		SwarmKey:                  n.swarmKey,
		UpdateStrategy:            g.opts.UpdateStrategy,
	}); err != nil {
		return fmt.Errorf("k8s create node %s: %w", name, err)
	}

	return
}

// DeleteNode deletes node from the k8s cluster and removes it from the node group
func (g *NodeGroup) DeleteNode(ctx context.Context, name string) (err error) {
	if err := g.k8s.Delete(ctx, name, g.cluster.namespace); err != nil {
		return fmt.Errorf("deleting node %s: %w", name, err)
	}

	g.deleteNode(name)

	return
}

// GroupReplicationFactor returns the total number of nodes in the node group that contain given chunk
func (g *NodeGroup) GroupReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error) {
	stream, err := g.HasChunkStream(ctx, a)
	if err != nil {
		return 0, fmt.Errorf("has chunk stream: %w", err)
	}

	var msgs []HasChunkStreamMsg
	for m := range stream {
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

// HasChunkStreamMsg represents message sent over the HasChunkStream channel
type HasChunkStreamMsg struct {
	Name  string
	Found bool
	Error error
}

// HasChunkStream returns stream of HasChunk requests for all nodes in the node group
func (g *NodeGroup) HasChunkStream(ctx context.Context, a swarm.Address) (<-chan HasChunkStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != k8s.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	hasChunkStream := make(chan HasChunkStreamMsg)
	go func() {
		var wg sync.WaitGroup
		for k, v := range g.nodes {
			if contains(stopped, v.name) {
				continue
			}

			wg.Add(1)
			go func(n string, c *Client) {
				defer wg.Done()

				found, err := c.HasChunk(ctx, a)
				hasChunkStream <- HasChunkStreamMsg{
					Name:  n,
					Found: found,
					Error: err,
				}
			}(k, v.client)
		}

		wg.Wait()
		close(hasChunkStream)
	}()

	return hasChunkStream, nil
}

// Name returns name of the node group
func (g *NodeGroup) Name() string {
	return g.name
}

// Nodes returns map of nodes in the node group
func (g *NodeGroup) Nodes() map[string]*Node {
	return g.getNodes()
}

// NodesClients returns map of node's clients in the node group excluding stopped nodes
func (g *NodeGroup) NodesClients(ctx context.Context) (map[string]*Client, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	clients := g.getClients()
	for _, n := range stopped {
		delete(clients, n)
	}

	return clients, nil
}

// NodesClientsAll returns map of node's clients in the node group
func (g *NodeGroup) NodesClientsAll(ctx context.Context) map[string]*Client {
	return g.getClients()
}

// NodesSorted returns sorted list of node names in the node group
func (g *NodeGroup) NodesSorted() (l []string) {
	nodes := g.getNodes()
	l = make([]string, len(nodes))

	i := 0
	for k := range g.nodes {
		l[i] = k
		i++
	}

	sort.Strings(l)

	return
}

// Node returns node
func (g *NodeGroup) Node(name string) *Node {
	return g.getNode(name)
}

// NodeClient returns node's client
func (g *NodeGroup) NodeClient(name string) *Client {
	return g.getClient(name)
}

// NodeGroupOverlays represents overlay addresses of all nodes in the node group
type NodeGroupOverlays map[string]swarm.Address

// Overlays returns NodeGroupOverlays
func (g *NodeGroup) Overlays(ctx context.Context) (overlays NodeGroupOverlays, err error) {
	stream, err := g.OverlaysStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("overlay stream: %w", err)
	}

	var msgs []OverlaysStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	overlays = make(NodeGroupOverlays)
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
func (g *NodeGroup) OverlaysStream(ctx context.Context) (<-chan OverlaysStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != k8s.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	overlaysStream := make(chan OverlaysStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.name) {
			continue
		}

		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Overlay(ctx)
			overlaysStream <- OverlaysStreamMsg{
				Name:    n,
				Address: a,
				Error:   err,
			}
		}(k, v.client)
	}

	go func() {
		wg.Wait()
		close(overlaysStream)
	}()

	return overlaysStream, nil
}

// NodeGroupPeers represents peers of all nodes in the node group
type NodeGroupPeers map[string][]swarm.Address

// Peers returns NodeGroupPeers
func (g *NodeGroup) Peers(ctx context.Context) (peers NodeGroupPeers, err error) {
	stream, err := g.PeersStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("peers stream: %w", err)
	}

	var msgs []PeersStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	peers = make(NodeGroupPeers)
	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
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
func (g *NodeGroup) PeersStream(ctx context.Context) (<-chan PeersStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != k8s.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	peersStream := make(chan PeersStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.name) {
			continue
		}

		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			a, err := c.Peers(ctx)
			peersStream <- PeersStreamMsg{
				Name:  n,
				Peers: a,
				Error: err,
			}
		}(k, v.client)
	}

	go func() {
		wg.Wait()
		close(peersStream)
	}()

	return peersStream, nil
}

// NodeReady returns node's readiness
func (g *NodeGroup) NodeReady(ctx context.Context, name string) (ok bool, err error) {
	return g.k8s.Ready(ctx, name, g.cluster.namespace)
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
	stream, err := g.SettlementsStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("settlements stream: %w", err)
	}

	overlays, err := g.Overlays(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking settlements: %w", err)
	}

	var msgs []SettlementsStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	settlements = make(NodeGroupSettlements)
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
func (g *NodeGroup) SettlementsStream(ctx context.Context) (<-chan SettlementsStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != k8s.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	SettlementsStream := make(chan SettlementsStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.name) {
			continue
		}

		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			s, err := c.Settlements(ctx)
			SettlementsStream <- SettlementsStreamMsg{
				Name:        n,
				Settlements: s,
				Error:       err,
			}
		}(k, v.client)
	}

	go func() {
		wg.Wait()
		close(SettlementsStream)
	}()

	return SettlementsStream, nil
}

// Size returns size of the node group
func (g *NodeGroup) Size() int {
	return len(g.nodes)
}

// StartNode start node by scaling its statefulset to 1
func (g *NodeGroup) StartNode(ctx context.Context, name string) (err error) {
	if err := g.k8s.Start(ctx, name, g.cluster.namespace); err != nil {
		return fmt.Errorf("start node %s: %w", name, err)
	}

	fmt.Printf("wait for %s to become ready\n", name)
	for {
		ok, err := g.NodeReady(ctx, name)
		if err != nil {
			return fmt.Errorf("node %s readiness: %w", name, err)
		}

		if ok {
			fmt.Printf("%s is ready\n", name)
			return nil
		}

		fmt.Printf("%s is not ready yet\n", name)
		time.Sleep(nodeReadyTimeout)
	}
}

// StartedNodes returns list of started nodes
func (g *NodeGroup) StartedNodes(ctx context.Context) (started []string, err error) {
	allStarted, err := g.k8s.StartedNodes(ctx, g.cluster.namespace)
	if err != nil {
		return nil, fmt.Errorf("k8s started nodes: %w", err)
	}

	for _, v := range allStarted {
		if contains(g.NodesSorted(), v) {
			started = append(started, v)
		}
	}

	return
}

// StopNode stops node by scaling down its statefulset to 0
func (g *NodeGroup) StopNode(ctx context.Context, name string) (err error) {
	return g.k8s.Stop(ctx, name, g.cluster.namespace)
}

// StoppedNodes returns list of stopped nodes
func (g *NodeGroup) StoppedNodes(ctx context.Context) (stopped []string, err error) {
	allStopped, err := g.k8s.StoppedNodes(ctx, g.cluster.namespace)
	if err != nil {
		return nil, fmt.Errorf("k8s stopped nodes: %w", err)
	}

	for _, v := range allStopped {
		if contains(g.NodesSorted(), v) {
			stopped = append(stopped, v)
		}
	}

	return
}

// NodeGroupTopologies represents Kademlia topology of all nodes in the node group
type NodeGroupTopologies map[string]Topology

// Topologies returns NodeGroupTopologies
func (g *NodeGroup) Topologies(ctx context.Context) (topologies NodeGroupTopologies, err error) {
	stream, err := g.TopologyStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("topology stream: %w", err)
	}

	var msgs []TopologyStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	topologies = make(NodeGroupTopologies)
	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
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

// TopologyStream returns stream of Kademlia topologies of all nodes in the node group
func (g *NodeGroup) TopologyStream(ctx context.Context) (<-chan TopologyStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != k8s.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	topologyStream := make(chan TopologyStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.name) {
			continue
		}

		wg.Add(1)
		go func(n string, c *Client) {
			defer wg.Done()

			t, err := c.Topology(ctx)
			topologyStream <- TopologyStreamMsg{
				Name:     n,
				Topology: t,
				Error:    err,
			}
		}(k, v.client)
	}

	go func() {
		wg.Wait()
		close(topologyStream)
	}()

	return topologyStream, nil
}

func (g *NodeGroup) addNode(n *Node) {
	g.lock.Lock()
	g.nodes[n.Name()] = n
	g.lock.Unlock()
}

func (g *NodeGroup) deleteNode(name string) {
	g.lock.Lock()
	delete(g.nodes, name)
	g.lock.Unlock()
}

func (g *NodeGroup) getClient(name string) *Client {
	g.lock.RLock()
	c := g.nodes[name].client
	g.lock.RUnlock()
	return c
}

func (g *NodeGroup) getClients() map[string]*Client {
	c := make(map[string]*Client)
	g.lock.RLock()
	for k, v := range g.nodes {
		c[k] = v.client
	}
	g.lock.RUnlock()
	return c
}

func (g *NodeGroup) getNode(name string) *Node {
	g.lock.RLock()
	n := g.nodes[name]
	g.lock.RUnlock()
	return n
}

func (g *NodeGroup) getNodes() map[string]*Node {
	g.lock.RLock()
	nodes := g.nodes
	g.lock.RUnlock()
	return nodes
}

func contains(list []string, find string) bool {
	for _, v := range list {
		if v == find {
			return true
		}
	}

	return false
}

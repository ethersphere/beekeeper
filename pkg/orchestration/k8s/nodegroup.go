package k8s

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

const nodeRetryTimeout = 3 * time.Second

// compile check whether client implements interface
var _ orchestration.NodeGroup = (*NodeGroup)(nil)

// NodeGroup represents group of Bee nodes
type NodeGroup struct {
	name  string
	nodes map[string]orchestration.Node
	opts  orchestration.NodeGroupOptions

	// set when added to the cluster
	cluster *Cluster
	k8s     orchestration.Bee

	lock sync.RWMutex
}

// NewNodeGroup returns new node group
func NewNodeGroup(name string, o orchestration.NodeGroupOptions) *NodeGroup {
	return &NodeGroup{
		name:  name,
		nodes: make(map[string]orchestration.Node),
		opts:  o,
	}
}

// AddNode adss new node to the node group
func (g *NodeGroup) AddNode(name string, o orchestration.NodeOptions) (err error) {
	aURL, err := g.cluster.apiURL(name)
	if err != nil {
		return fmt.Errorf("API URL %s: %w", name, err)
	}

	dURL, err := g.cluster.debugAPIURL(name)
	if err != nil {
		return fmt.Errorf("debug API URL %s: %w", name, err)
	}

	client := bee.NewClient(bee.ClientOptions{
		APIURL:              aURL,
		APIInsecureTLS:      g.cluster.apiInsecureTLS,
		DebugAPIURL:         dURL,
		DebugAPIInsecureTLS: g.cluster.debugAPIInsecureTLS,
		Retry:               5,
	})

	// TODO: make more granular, check every sub-option
	var config *orchestration.Config
	if o.Config != nil {
		config = o.Config
	} else {
		config = g.opts.BeeConfig
	}

	n := NewNode(name, orchestration.NodeOptions{
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

// Addresses returns NodeGroupAddresses
func (g *NodeGroup) Addresses(ctx context.Context) (addrs orchestration.NodeGroupAddresses, err error) {
	stream, err := g.AddressesStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("addresses stream: %w", err)
	}

	var msgs []AddressesStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	addrs = make(orchestration.NodeGroupAddresses)
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
	Addresses bee.Addresses
	Error     error
}

// AddressesStream returns stream of addresses of all nodes in the node group
func (g *NodeGroup) AddressesStream(ctx context.Context) (<-chan AddressesStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != orchestration.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	addressStream := make(chan AddressesStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.Name()) {
			continue
		}

		wg.Add(1)
		go func(n string, c *bee.Client) {
			defer wg.Done()

			a, err := c.Addresses(ctx)
			addressStream <- AddressesStreamMsg{
				Name:      n,
				Addresses: a,
				Error:     err,
			}
		}(k, v.Client())
	}

	go func() {
		wg.Wait()
		close(addressStream)
	}()

	return addressStream, nil
}

// Balances returns NodeGroupBalances
func (g *NodeGroup) Balances(ctx context.Context) (balances orchestration.NodeGroupBalances, err error) {
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

	balances = make(orchestration.NodeGroupBalances)
	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
		}

		tmp := make(map[string]int64)
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
	Balances bee.Balances
	Error    error
}

// BalancesStream returns stream of balances of all nodes in the cluster
func (g *NodeGroup) BalancesStream(ctx context.Context) (<-chan BalancesStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != orchestration.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	balancesStream := make(chan BalancesStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.Name()) {
			continue
		}

		wg.Add(1)
		go func(n string, c *bee.Client) {
			defer wg.Done()

			b, err := c.Balances(ctx)
			balancesStream <- BalancesStreamMsg{
				Name:     n,
				Balances: b,
				Error:    err,
			}
		}(k, v.Client())
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

	n, err := g.getNode(name)
	if err != nil {
		return err
	}

	if err := g.k8s.Create(ctx, orchestration.CreateOptions{
		// Bee configuration
		Config: *n.Config(),
		// Kubernetes configuration
		Name:                      name,
		Namespace:                 g.cluster.namespace,
		Annotations:               g.opts.Annotations,
		ClefImage:                 g.opts.ClefImage,
		ClefImagePullPolicy:       g.opts.ClefImagePullPolicy,
		ClefKey:                   n.ClefKey(),
		ClefPassword:              n.ClefPassword(),
		Image:                     g.opts.Image,
		ImagePullPolicy:           g.opts.ImagePullPolicy,
		ImagePullSecrets:          g.opts.ImagePullSecrets,
		IngressAnnotations:        g.opts.IngressAnnotations,
		IngressClass:              g.opts.IngressClass,
		IngressHost:               g.cluster.ingressHost(name),
		IngressDebugAnnotations:   g.opts.IngressDebugAnnotations,
		IngressDebugClass:         g.opts.IngressDebugClass,
		IngressDebugHost:          g.cluster.ingressDebugHost(name),
		Labels:                    labels,
		LibP2PKey:                 n.LibP2PKey(),
		NodeSelector:              g.opts.NodeSelector,
		PersistenceEnabled:        g.opts.PersistenceEnabled,
		PersistenceStorageClass:   g.opts.PersistenceStorageClass,
		PersistenceStorageRequest: g.opts.PersistenceStorageRequest,
		PodManagementPolicy:       g.opts.PodManagementPolicy,
		RestartPolicy:             g.opts.RestartPolicy,
		ResourcesLimitCPU:         g.opts.ResourcesLimitCPU,
		ResourcesLimitMemory:      g.opts.ResourcesLimitMemory,
		ResourcesRequestCPU:       g.opts.ResourcesRequestCPU,
		ResourcesRequestMemory:    g.opts.ResourcesRequestMemory,
		Selector:                  labels,
		SwarmKey:                  n.SwarmKey(),
		UpdateStrategy:            g.opts.UpdateStrategy,
	}); err != nil {
		return err
	}

	return
}

// DeleteNode deletes node from the k8s cluster and removes it from the node group
func (g *NodeGroup) DeleteNode(ctx context.Context, name string) (err error) {
	if err := g.k8s.Delete(ctx, name, g.cluster.namespace); err != nil {
		return err
	}

	g.deleteNode(name)

	return
}

// Fund adds funds to the node
func (g *NodeGroup) Fund(ctx context.Context, name string, o orchestration.FundingOptions) (err error) {
	var a bee.Addresses
	if o.Eth > 0 || o.Bzz > 0 || o.GBzz > 0 {
		retries := 5
		for {
			c, err := g.NodeClient(name)
			if err != nil {
				return err
			}
			a, err = c.Addresses(ctx)
			if err != nil {
				retries--
				if retries == 0 {
					return fmt.Errorf("get %s address: %w", name, err)
				}
				time.Sleep(nodeRetryTimeout)
				continue
			}
			break
		}
	}

	if o.Eth > 0 {
		retries := 5
		for {
			tx, err := g.cluster.swap.SendETH(ctx, a.Ethereum, o.Eth)
			if err != nil {
				retries--
				if retries == 0 {
					return fmt.Errorf("send eth: %w", err)
				}
				time.Sleep(nodeRetryTimeout)
				continue
			}
			fmt.Printf("%s funded with %.2f ETH, transaction: %s\n", name, o.Eth, tx)
			break
		}
	}

	if o.Bzz > 0 {
		retries := 5
		for {
			tx, err := g.cluster.swap.SendBZZ(ctx, a.Ethereum, o.Bzz)
			if err != nil {
				retries--
				if retries == 0 {
					return fmt.Errorf("send eth: %w", err)
				}
				time.Sleep(nodeRetryTimeout)
				continue
			}
			fmt.Printf("%s funded with %.2f BZZ, transaction: %s\n", name, o.Bzz, tx)
			break
		}
	}

	if o.GBzz > 0 {
		retries := 5
		for {
			tx, err := g.cluster.swap.SendGBZZ(ctx, a.Ethereum, o.GBzz)
			if err != nil {
				retries--
				if retries == 0 {
					return fmt.Errorf("send eth: %w", err)
				}
				time.Sleep(nodeRetryTimeout)
				continue
			}
			fmt.Printf("%s funded with %.2f gBZZ, transaction: %s\n", name, o.GBzz, tx)
			break
		}
	}

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
	if err != nil && err != orchestration.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	hasChunkStream := make(chan HasChunkStreamMsg)
	go func() {
		var wg sync.WaitGroup
		for k, v := range g.nodes {
			if contains(stopped, v.Name()) {
				continue
			}

			wg.Add(1)
			go func(n string, c *bee.Client) {
				defer wg.Done()

				found, err := c.HasChunk(ctx, a)
				hasChunkStream <- HasChunkStreamMsg{
					Name:  n,
					Found: found,
					Error: err,
				}
			}(k, v.Client())
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
func (g *NodeGroup) Nodes() map[string]orchestration.Node {
	return g.getNodes()
}

// NodesClients returns map of node's clients in the node group excluding stopped nodes
func (g *NodeGroup) NodesClients(ctx context.Context) (map[string]*bee.Client, error) {
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
func (g *NodeGroup) NodesClientsAll(ctx context.Context) map[string]*bee.Client {
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
func (g *NodeGroup) Node(name string) (orchestration.Node, error) {
	return g.getNode(name)
}

// NodeClient returns node's client
func (g *NodeGroup) NodeClient(name string) (*bee.Client, error) {
	return g.getClient(name)
}

// Overlays returns NodeGroupOverlays
func (g *NodeGroup) Overlays(ctx context.Context) (overlays orchestration.NodeGroupOverlays, err error) {
	stream, err := g.OverlaysStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("overlay stream: %w", err)
	}

	var msgs []OverlaysStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	overlays = make(orchestration.NodeGroupOverlays)
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
	if err != nil && err != orchestration.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	overlaysStream := make(chan OverlaysStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.Name()) {
			continue
		}

		wg.Add(1)
		go func(n string, c *bee.Client) {
			defer wg.Done()

			a, err := c.Overlay(ctx)
			overlaysStream <- OverlaysStreamMsg{
				Name:    n,
				Address: a,
				Error:   err,
			}
		}(k, v.Client())
	}

	go func() {
		wg.Wait()
		close(overlaysStream)
	}()

	return overlaysStream, nil
}

// Peers returns NodeGroupPeers
func (g *NodeGroup) Peers(ctx context.Context) (peers orchestration.NodeGroupPeers, err error) {
	stream, err := g.PeersStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("peers stream: %w", err)
	}

	var msgs []PeersStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	peers = make(orchestration.NodeGroupPeers)
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
	if err != nil && err != orchestration.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	peersStream := make(chan PeersStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.Name()) {
			continue
		}

		wg.Add(1)
		go func(n string, c *bee.Client) {
			defer wg.Done()

			a, err := c.Peers(ctx)
			peersStream <- PeersStreamMsg{
				Name:  n,
				Peers: a,
				Error: err,
			}
		}(k, v.Client())
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

// RunningNodes returns list of running nodes
func (g *NodeGroup) RunningNodes(ctx context.Context) (running []string, err error) {
	allRunning, err := g.k8s.RunningNodes(ctx, g.cluster.namespace)
	if err != nil {
		return nil, err
	}

	for _, v := range allRunning {
		if contains(g.NodesSorted(), v) {
			running = append(running, v)
		}
	}

	return
}

// SetupNode creates new node in the node group, starts it in the k8s cluster and funds it
func (g *NodeGroup) SetupNode(ctx context.Context, name string, o orchestration.NodeOptions, f orchestration.FundingOptions) (err error) {
	if err := g.AddNode(name, o); err != nil {
		return fmt.Errorf("add node %s: %w", name, err)
	}

	if err := g.CreateNode(ctx, name); err != nil {
		return fmt.Errorf("create node %s in k8s: %w", name, err)
	}

	if err := g.StartNode(ctx, name); err != nil {
		return fmt.Errorf("start node %s in k8s: %w", name, err)
	}

	if err := g.Fund(ctx, name, f); err != nil {
		return fmt.Errorf("fund node %s: %w", name, err)
	}

	return
}

// Settlements returns NodeGroupSettlements
func (g *NodeGroup) Settlements(ctx context.Context) (settlements orchestration.NodeGroupSettlements, err error) {
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

	settlements = make(orchestration.NodeGroupSettlements)
	for _, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %w", m.Name, m.Error)
		}

		tmp := make(map[string]orchestration.SentReceived)
		for _, s := range m.Settlements.Settlements {
			tmp[s.Peer] = orchestration.SentReceived{
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
	Settlements bee.Settlements
	Error       error
}

// SettlementsStream returns stream of settlements of all nodes in the cluster
func (g *NodeGroup) SettlementsStream(ctx context.Context) (<-chan SettlementsStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != orchestration.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	SettlementsStream := make(chan SettlementsStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.Name()) {
			continue
		}

		wg.Add(1)
		go func(n string, c *bee.Client) {
			defer wg.Done()

			s, err := c.Settlements(ctx)
			SettlementsStream <- SettlementsStreamMsg{
				Name:        n,
				Settlements: s,
				Error:       err,
			}
		}(k, v.Client())
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
		return err
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
		time.Sleep(nodeRetryTimeout)
	}
}

// StopNode stops node by scaling down its statefulset to 0
func (g *NodeGroup) StopNode(ctx context.Context, name string) (err error) {
	if err := g.k8s.Stop(ctx, name, g.cluster.namespace); err != nil {
		return err
	}

	fmt.Printf("wait for %s to stop\n", name)
	for {
		ok, err := g.NodeReady(ctx, name)
		if err != nil {
			return fmt.Errorf("node %s readiness: %w", name, err)
		}

		if !ok {
			fmt.Printf("%s is stopped\n", name)
			return nil
		}

		fmt.Printf("%s is not stopped yet\n", name)
		time.Sleep(nodeRetryTimeout)
	}
}

// StoppedNodes returns list of stopped nodes
func (g *NodeGroup) StoppedNodes(ctx context.Context) (stopped []string, err error) {
	allStopped, err := g.k8s.StoppedNodes(ctx, g.cluster.namespace)
	if err != nil {
		return nil, err
	}

	for _, v := range allStopped {
		if contains(g.NodesSorted(), v) {
			stopped = append(stopped, v)
		}
	}

	return
}

// Topologies returns NodeGroupTopologies
func (g *NodeGroup) Topologies(ctx context.Context) (topologies orchestration.NodeGroupTopologies, err error) {
	stream, err := g.TopologyStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("topology stream: %w", err)
	}

	var msgs []TopologyStreamMsg
	for m := range stream {
		msgs = append(msgs, m)
	}

	topologies = make(orchestration.NodeGroupTopologies)
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
	Topology bee.Topology
	Error    error
}

// TopologyStream returns stream of Kademlia topologies of all nodes in the node group
func (g *NodeGroup) TopologyStream(ctx context.Context) (<-chan TopologyStreamMsg, error) {
	stopped, err := g.StoppedNodes(ctx)
	if err != nil && err != orchestration.ErrNotSet {
		return nil, fmt.Errorf("stopped nodes: %w", err)
	}

	topologyStream := make(chan TopologyStreamMsg)
	var wg sync.WaitGroup
	for k, v := range g.nodes {
		if contains(stopped, v.Name()) {
			continue
		}

		wg.Add(1)
		go func(n string, c *bee.Client) {
			defer wg.Done()

			t, err := c.Topology(ctx)
			topologyStream <- TopologyStreamMsg{
				Name:     n,
				Topology: t,
				Error:    err,
			}
		}(k, v.Client())
	}

	go func() {
		wg.Wait()
		close(topologyStream)
	}()

	return topologyStream, nil
}

func (g *NodeGroup) addNode(n orchestration.Node) {
	g.lock.Lock()
	g.nodes[n.Name()] = n
	g.lock.Unlock()
}

func (g *NodeGroup) deleteNode(name string) {
	g.lock.Lock()
	delete(g.nodes, name)
	g.lock.Unlock()
}

func (g *NodeGroup) getClient(name string) (*bee.Client, error) {
	n, err := g.getNode(name)
	if err != nil {
		return nil, err
	}
	return n.Client(), nil
}

func (g *NodeGroup) getClients() map[string]*bee.Client {
	c := make(map[string]*bee.Client)
	g.lock.RLock()
	for k, v := range g.nodes {
		c[k] = v.Client()
	}
	g.lock.RUnlock()
	return c
}

func (g *NodeGroup) getNode(name string) (n orchestration.Node, err error) {
	g.lock.RLock()
	n, ok := g.nodes[name]
	g.lock.RUnlock()
	if !ok {
		return Node{}, fmt.Errorf("node %s not found", name)
	}
	return
}

func (g *NodeGroup) getNodes() map[string]orchestration.Node {
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

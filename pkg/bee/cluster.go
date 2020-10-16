package bee

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"sync"

	"github.com/ethersphere/bee/pkg/swarm"
)

// Cluster represents cluster of Bee nodes
type Cluster struct {
	Nodes []Client
	opts  ClusterOptions
}

// ClusterOptions represents Bee cluster options
type ClusterOptions struct {
	APIScheme               string
	APIHostnamePattern      string
	APIDomain               string
	APIInsecureTLS          bool
	DebugAPIScheme          string
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	DisableNamespace        bool
	DebugAPIInsecureTLS     bool
	Namespace               string
	Size                    int
}

// NewCluster returns new cluster
func NewCluster(o ClusterOptions) (c Cluster, err error) {
	c.opts = o
	for i := 0; i < o.Size; i++ {
		a, err := createURL(o.APIScheme, o.APIHostnamePattern, o.Namespace, o.APIDomain, o.DisableNamespace, i)
		if err != nil {
			return Cluster{}, fmt.Errorf("create cluster: %w", err)
		}

		d, err := createURL(o.DebugAPIScheme, o.DebugAPIHostnamePattern, o.Namespace, o.DebugAPIDomain, o.DisableNamespace, i)
		if err != nil {
			return Cluster{}, fmt.Errorf("create cluster: %w", err)
		}

		n := NewClient(ClientOptions{
			APIURL:              a,
			APIInsecureTLS:      o.APIInsecureTLS,
			DebugAPIURL:         d,
			DebugAPIInsecureTLS: o.DebugAPIInsecureTLS,
		})

		c.Nodes = append(c.Nodes, n)
	}

	return
}

// AddNodes adds new nodes to the cluster
func (cs *Cluster) AddNodes(count int) (err error) {
	start, stop := cs.Size(), cs.Size()+count

	for i := start; i < stop; i++ {
		a, err := createURL(cs.opts.APIScheme, cs.opts.APIHostnamePattern, cs.opts.Namespace, cs.opts.APIDomain, cs.opts.DisableNamespace, i)
		if err != nil {
			return fmt.Errorf("add nodes: %w", err)
		}

		d, err := createURL(cs.opts.DebugAPIScheme, cs.opts.DebugAPIHostnamePattern, cs.opts.Namespace, cs.opts.DebugAPIDomain, cs.opts.DisableNamespace, i)
		if err != nil {
			return fmt.Errorf("add nodes: %w", err)
		}

		n := NewClient(ClientOptions{
			APIURL:              a,
			APIInsecureTLS:      cs.opts.APIInsecureTLS,
			DebugAPIURL:         d,
			DebugAPIInsecureTLS: cs.opts.DebugAPIInsecureTLS,
		})

		cs.Nodes = append(cs.Nodes, n)
	}

	return
}

// Addresses returns ordered list of addresses of all nodes in the cluster
func (cs *Cluster) Addresses(ctx context.Context) (addrs []Addresses, err error) {
	var msgs []AddressesStreamMsg
	for m := range cs.AddressesStream(ctx) {
		msgs = append(msgs, m)
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Index < msgs[j].Index
	})

	for i, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("node %d: %w", i, m.Error)
		}
		addrs = append(addrs, m.Addresses)
	}

	return
}

// AddressesStreamMsg represents message sent over the AddressStream channel
type AddressesStreamMsg struct {
	Addresses Addresses
	Index     int
	Error     error
}

// AddressesStream returns stream of addresses of all nodes in the cluster
func (cs *Cluster) AddressesStream(ctx context.Context) <-chan AddressesStreamMsg {
	addressStream := make(chan AddressesStreamMsg)

	var wg sync.WaitGroup
	for i, node := range cs.Nodes {
		wg.Add(1)
		go func(i int, n Client) {
			defer wg.Done()

			a, err := n.Addresses(ctx)
			addressStream <- AddressesStreamMsg{
				Addresses: a,
				Index:     i,
				Error:     err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(addressStream)
	}()

	return addressStream
}

// Balances returns balances of all nodes in the cluster
func (cs *Cluster) Balances(ctx context.Context) (balances map[string]map[string]int, err error) {
	overlays, err := cs.Overlays(ctx)
	if err != nil {
		return nil, err
	}

	var msgs []BalancesStreamMsg
	for m := range cs.BalancesStream(ctx) {
		msgs = append(msgs, m)
	}
	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Index < msgs[j].Index
	})

	balances = make(map[string]map[string]int)
	for i, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("node %d: %w", i, m.Error)
		}

		tmp := make(map[string]int)
		for _, b := range m.Balances.Balances {
			tmp[b.Peer] = b.Balance
		}
		balances[overlays[i].String()] = tmp
	}

	return
}

// BalancesStreamMsg represents message sent over the BalancesStream channel
type BalancesStreamMsg struct {
	Balances Balances
	Index    int
	Error    error
}

// BalancesStream returns stream of balances of all nodes in the cluster
func (cs *Cluster) BalancesStream(ctx context.Context) <-chan BalancesStreamMsg {
	balancesStream := make(chan BalancesStreamMsg)

	var wg sync.WaitGroup
	for i, node := range cs.Nodes {
		wg.Add(1)
		go func(i int, n Client) {
			defer wg.Done()

			b, err := n.Balances(ctx)
			balancesStream <- BalancesStreamMsg{
				Balances: b,
				Index:    i,
				Error:    err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(balancesStream)
	}()

	return balancesStream
}

// GlobalReplicationFactor returns the total number of nodes that contain given chunk
func (cs *Cluster) GlobalReplicationFactor(ctx context.Context, a swarm.Address) (int, error) {
	var counter int
	for m := range cs.HasChunkStream(ctx, a) {
		if m.Error != nil {
			return 0, fmt.Errorf("node %d: %w", m.Index, m.Error)
		}
		if m.Found {
			counter++
		}
	}

	return counter, nil
}

// HasChunkStreamMsg represents message sent over the HasChunkStream channel
type HasChunkStreamMsg struct {
	Found bool
	Index int
	Error error
}

// HasChunkStream returns stream of HasChunk requests for all nodes in the cluster
func (cs *Cluster) HasChunkStream(ctx context.Context, a swarm.Address) <-chan HasChunkStreamMsg {
	hasChunkStream := make(chan HasChunkStreamMsg)

	go func() {
		var wg sync.WaitGroup
		for i, node := range cs.Nodes {
			wg.Add(1)
			go func(i int, n Client) {
				defer wg.Done()

				found, err := n.HasChunk(ctx, a)
				hasChunkStream <- HasChunkStreamMsg{
					Found: found,
					Index: i,
					Error: err,
				}
			}(i, node)
		}

		wg.Wait()
		close(hasChunkStream)
	}()

	return hasChunkStream
}

// Overlays returns ordered list of overlay addresses of all nodes in the cluster
func (cs *Cluster) Overlays(ctx context.Context) (overlays []swarm.Address, err error) {
	var msgs []OverlaysStreamMsg
	for m := range cs.OverlaysStream(ctx) {
		msgs = append(msgs, m)
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Index < msgs[j].Index
	})

	for i, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("node %d: %w", i, m.Error)
		}
		overlays = append(overlays, m.Address)
	}

	return
}

// OverlaysStreamMsg represents message sent over the OverlaysStream channel
type OverlaysStreamMsg struct {
	Address swarm.Address
	Index   int
	Error   error
}

// OverlaysStream returns stream of overlay addresses of all nodes in the cluster
// TODO: add semaphore
func (cs *Cluster) OverlaysStream(ctx context.Context) <-chan OverlaysStreamMsg {
	overlaysStream := make(chan OverlaysStreamMsg)

	var wg sync.WaitGroup
	for i, node := range cs.Nodes {
		wg.Add(1)
		go func(i int, n Client) {
			defer wg.Done()

			a, err := n.Overlay(ctx)
			overlaysStream <- OverlaysStreamMsg{
				Address: a,
				Index:   i,
				Error:   err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(overlaysStream)
	}()

	return overlaysStream
}

// Peers returns ordered list of peers of all nodes in the cluster
func (cs *Cluster) Peers(ctx context.Context) (peers [][]swarm.Address, err error) {
	var msgs []PeersStreamMsg
	for m := range cs.PeersStream(ctx) {
		msgs = append(msgs, m)
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Index < msgs[j].Index
	})

	for i, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("node %d: %w", i, m.Error)
		}
		peers = append(peers, m.Peers)
	}

	return
}

// PeersStreamMsg represents message sent over the PeersStream channel
type PeersStreamMsg struct {
	Peers []swarm.Address
	Index int
	Error error
}

// PeersStream returns stream of peers of all nodes in the cluster
func (cs *Cluster) PeersStream(ctx context.Context) <-chan PeersStreamMsg {
	peersStream := make(chan PeersStreamMsg)

	var wg sync.WaitGroup
	for i, node := range cs.Nodes {
		wg.Add(1)
		go func(i int, n Client) {
			defer wg.Done()

			a, err := n.Peers(ctx)
			peersStream <- PeersStreamMsg{
				Peers: a,
				Index: i,
				Error: err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(peersStream)
	}()

	return peersStream
}

// RemoveNodes removes nodes from the cluster
func (cs *Cluster) RemoveNodes(count int) (err error) {
	cs.Nodes = cs.Nodes[:cs.Size()-count]

	return
}

// SentReceived object
type SentReceived struct {
	Received int
	Sent     int
}

// Settlements returns settlements of all nodes in the cluster
func (cs *Cluster) Settlements(ctx context.Context) (settlements map[string]map[string]SentReceived, err error) {
	overlays, err := cs.Overlays(ctx)
	if err != nil {
		return nil, err
	}

	var msgs []SettlementsStreamMsg
	for m := range cs.SettlementsStream(ctx) {
		msgs = append(msgs, m)
	}
	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Index < msgs[j].Index
	})

	settlements = make(map[string]map[string]SentReceived)
	for i, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("node %d: %w", i, m.Error)
		}

		tmp := make(map[string]SentReceived)
		for _, s := range m.Settlements.Settlements {
			tmp[s.Peer] = SentReceived{
				Received: s.Received,
				Sent:     s.Sent,
			}
		}
		settlements[overlays[i].String()] = tmp
	}

	return
}

// SettlementsStreamMsg represents message sent over the SettlementsStream channel
type SettlementsStreamMsg struct {
	Settlements Settlements
	Index       int
	Error       error
}

// SettlementsStream returns stream of settlements of all nodes in the cluster
func (cs *Cluster) SettlementsStream(ctx context.Context) <-chan SettlementsStreamMsg {
	SettlementsStream := make(chan SettlementsStreamMsg)

	var wg sync.WaitGroup
	for i, node := range cs.Nodes {
		wg.Add(1)
		go func(i int, n Client) {
			defer wg.Done()

			s, err := n.Settlements(ctx)
			SettlementsStream <- SettlementsStreamMsg{
				Settlements: s,
				Index:       i,
				Error:       err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(SettlementsStream)
	}()

	return SettlementsStream
}

// Size returns size of the cluster
func (cs *Cluster) Size() int {
	return len(cs.Nodes)
}

// Topologies returns ordered list of Kademlia topology of all nodes in the cluster
func (cs *Cluster) Topologies(ctx context.Context) (topologies []Topology, err error) {
	var msgs []TopologyStreamMsg
	for m := range cs.TopologyStream(ctx) {
		msgs = append(msgs, m)
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Index < msgs[j].Index
	})

	for i, m := range msgs {
		if m.Error != nil {
			return nil, fmt.Errorf("node %d: %w", i, m.Error)
		}
		topologies = append(topologies, m.Topology)
	}

	return
}

// TopologyStreamMsg represents message sent over the TopologyStream channel
type TopologyStreamMsg struct {
	Topology Topology
	Index    int
	Error    error
}

// TopologyStream returns stream of peers of all nodes in the cluster
func (cs *Cluster) TopologyStream(ctx context.Context) <-chan TopologyStreamMsg {
	topologyStream := make(chan TopologyStreamMsg)

	var wg sync.WaitGroup
	for i, node := range cs.Nodes {
		wg.Add(1)
		go func(i int, n Client) {
			defer wg.Done()

			t, err := n.Topology(ctx)
			topologyStream <- TopologyStreamMsg{
				Topology: t,
				Index:    i,
				Error:    err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(topologyStream)
	}()

	return topologyStream
}

// createURL creates API or debug API URL
func createURL(scheme, hostnamePattern, namespace, domain string, disableNamespace bool, index int) (nodeURL *url.URL, err error) {
	hostname := fmt.Sprintf(hostnamePattern, index)
	if disableNamespace {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s", scheme, hostname, domain))
	} else {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", scheme, hostname, namespace, domain))
	}
	return
}

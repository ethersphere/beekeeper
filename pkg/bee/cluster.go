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
	Nodes []Node
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
	DebugAPIInsecureTLS     bool
	Namespace               string
	Size                    int
}

// NewCluster returns new cluster
func NewCluster(o ClusterOptions) (c Cluster, err error) {
	for i := 0; i < o.Size; i++ {
		a, err := createURL(o.APIScheme, o.APIHostnamePattern, o.Namespace, o.APIDomain, i)
		if err != nil {
			return Cluster{}, fmt.Errorf("create cluster: %w", err)
		}

		d, err := createURL(o.DebugAPIScheme, o.DebugAPIHostnamePattern, o.Namespace, o.DebugAPIDomain, i)
		if err != nil {
			return Cluster{}, fmt.Errorf("create cluster: %w", err)
		}

		n := NewNode(NodeOptions{
			APIURL:              a,
			APIInsecureTLS:      o.APIInsecureTLS,
			DebugAPIURL:         d,
			DebugAPIInsecureTLS: o.DebugAPIInsecureTLS,
		})

		c.Nodes = append(c.Nodes, n)
	}

	return
}

// Addresses returns ordered list of addresses of all nodes in the cluster
func (c *Cluster) Addresses(ctx context.Context) (addrs []Addresses, err error) {
	var msgs []AddressesStreamMsg
	for m := range c.AddressesStream(ctx) {
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
func (c *Cluster) AddressesStream(ctx context.Context) <-chan AddressesStreamMsg {
	addressStream := make(chan AddressesStreamMsg)

	var wg sync.WaitGroup
	for i, node := range c.Nodes {
		wg.Add(1)
		go func(i int, n Node) {
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

// ChunkReplicationFactor returns replication factor for a given chunk
func (c *Cluster) ChunkReplicationFactor(ctx context.Context, a swarm.Address) (counter int, err error) {
	for m := range c.HasChunkStream(ctx, a) {
		if m.Error != nil {
			return 0, fmt.Errorf("node %d: %w", m.Index, m.Error)
		}
		if m.Found {
			counter++
		}
	}

	return
}

// HasChunkStreamMsg represents message sent over the HasChunkStream channel
type HasChunkStreamMsg struct {
	Found bool
	Index int
	Error error
}

// HasChunkStream returns stream of HasChunk requests for all nodes in the cluster
func (c *Cluster) HasChunkStream(ctx context.Context, a swarm.Address) <-chan HasChunkStreamMsg {
	HasChunkStream := make(chan HasChunkStreamMsg)

	var wg sync.WaitGroup
	for i, node := range c.Nodes {
		wg.Add(1)
		go func(i int, n Node) {
			defer wg.Done()

			found, err := n.HasChunk(ctx, a)
			HasChunkStream <- HasChunkStreamMsg{
				Found: found,
				Index: i,
				Error: err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(HasChunkStream)
	}()

	return HasChunkStream
}

// Overlays returns ordered list of overlay addresses of all nodes in the cluster
func (c *Cluster) Overlays(ctx context.Context) (overlays []swarm.Address, err error) {
	var msgs []OverlaysStreamMsg
	for m := range c.OverlaysStream(ctx) {
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
func (c *Cluster) OverlaysStream(ctx context.Context) <-chan OverlaysStreamMsg {
	overlaysStream := make(chan OverlaysStreamMsg)

	var wg sync.WaitGroup
	for i, node := range c.Nodes {
		wg.Add(1)
		go func(i int, n Node) {
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
func (c *Cluster) Peers(ctx context.Context) (peers [][]swarm.Address, err error) {
	var msgs []PeersStreamMsg
	for m := range c.PeersStream(ctx) {
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
func (c *Cluster) PeersStream(ctx context.Context) <-chan PeersStreamMsg {
	peersStream := make(chan PeersStreamMsg)

	var wg sync.WaitGroup
	for i, node := range c.Nodes {
		wg.Add(1)
		go func(i int, n Node) {
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

// Size returns size of the cluster
func (c *Cluster) Size() int {
	return len(c.Nodes)
}

// Topologies returns ordered list of Kademlia topology of all nodes in the cluster
func (c *Cluster) Topologies(ctx context.Context) (topologies []Topology, err error) {
	var msgs []TopologyStreamMsg
	for m := range c.TopologyStream(ctx) {
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
func (c *Cluster) TopologyStream(ctx context.Context) <-chan TopologyStreamMsg {
	topologyStream := make(chan TopologyStreamMsg)

	var wg sync.WaitGroup
	for i, node := range c.Nodes {
		wg.Add(1)
		go func(i int, n Node) {
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
func createURL(scheme, hostnamePattern, namespace, domain string, index int) (nodeURL *url.URL, err error) {
	hostname := fmt.Sprintf(hostnamePattern, index)
	if len(namespace) == 0 {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s", scheme, hostname, domain))
	} else {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", scheme, hostname, namespace, domain))
	}
	return
}

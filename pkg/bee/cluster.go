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
			return Cluster{}, err
		}

		d, err := createURL(o.DebugAPIScheme, o.DebugAPIHostnamePattern, o.Namespace, o.DebugAPIDomain, i)
		if err != nil {
			return Cluster{}, err
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

	for _, m := range msgs {
		if m.Error != nil {
			return []Addresses{}, m.Error
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

// Overlays returns ordered list of overlay addresses of all nodes in the cluster
func (c *Cluster) Overlays(ctx context.Context) (overlays []swarm.Address, err error) {
	var msgs []OverlaysStreamMsg
	for m := range c.OverlaysStream(ctx) {
		msgs = append(msgs, m)
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Index < msgs[j].Index
	})

	for _, m := range msgs {
		if m.Error != nil {
			return nil, m.Error
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
// TODO: add semaphor
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

// Size returns size of the cluster
func (c *Cluster) Size() int {
	return len(c.Nodes)
}

// Underlays returns ordered list of underlay addresses of all nodes in the cluster
func (c *Cluster) Underlays(ctx context.Context) (underlays [][]string, err error) {
	var msgs []UnderlaysStreamMsg
	for m := range c.UnderlaysStream(ctx) {
		msgs = append(msgs, m)
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Index < msgs[j].Index
	})

	for _, m := range msgs {
		if m.Error != nil {
			return [][]string{}, m.Error
		}
		underlays = append(underlays, m.Address)
	}

	return
}

// UnderlaysStreamMsg represents message sent over the UnderlaysStream channel
type UnderlaysStreamMsg struct {
	Address []string
	Index   int
	Error   error
}

// UnderlaysStream returns stream of underlay addresses of all nodes in the cluster
func (c *Cluster) UnderlaysStream(ctx context.Context) <-chan UnderlaysStreamMsg {
	underlaysStream := make(chan UnderlaysStreamMsg)

	var wg sync.WaitGroup
	for i, node := range c.Nodes {
		wg.Add(1)
		go func(i int, n Node) {
			defer wg.Done()

			a, err := n.Underlay(ctx)
			underlaysStream <- UnderlaysStreamMsg{
				Address: a,
				Index:   i,
				Error:   err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(underlaysStream)
	}()

	return underlaysStream
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

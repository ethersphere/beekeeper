package bee

import (
	"context"
	"fmt"
	"net/url"

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

// Addresses returns addresses of all nodes in the cluster
func (c *Cluster) Addresses(ctx context.Context) (resp []Addresses, err error) {
	for _, n := range c.Nodes {
		a, err := n.Addresses(ctx)
		if err != nil {
			return []Addresses{}, err
		}

		resp = append(resp, a)
	}

	return
}

// Overlays returns overlay addresses of all nodes in the cluster
func (c *Cluster) Overlays(ctx context.Context) (overlays []swarm.Address, err error) {
	for _, n := range c.Nodes {
		a, err := n.Overlay(ctx)
		if err != nil {
			return []swarm.Address{}, err
		}

		overlays = append(overlays, a)
	}

	return
}

// Size returns size of the cluster
func (c *Cluster) Size() int {
	return len(c.Nodes)
}

// Underlays returns underlay addresses of all nodes in the cluster
func (c *Cluster) Underlays(ctx context.Context) (underlays [][]string, err error) {
	for _, n := range c.Nodes {
		u, err := n.Underlay(ctx)
		if err != nil {
			return [][]string{}, err
		}

		underlays = append(underlays, u)
	}

	return
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

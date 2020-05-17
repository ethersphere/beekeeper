package bee

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ethersphere/bee/pkg/swarm"
)

const (
	scheme = "http"
)

// Cluster represents cluster of Bee nodes
type Cluster struct {
	Nodes []Node
}

// ClusterOptions represents Bee cluster options
type ClusterOptions struct {
	APIHostnamePattern      string
	APIDomain               string
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	Namespace               string
	Size                    int
}

// NewCluster returns new cluster
func NewCluster(o ClusterOptions) (cluster Cluster, err error) {
	for i := 0; i < o.Size; i++ {
		a, err := createURL(scheme, o.APIHostnamePattern, o.Namespace, o.APIDomain, i)
		if err != nil {
			return Cluster{}, err
		}

		d, err := createURL(scheme, o.DebugAPIHostnamePattern, o.Namespace, o.DebugAPIDomain, i)
		if err != nil {
			return Cluster{}, err
		}

		n := NewNode(NodeOptions{
			APIURL:   a,
			DebugURL: d,
		})

		cluster.Nodes = append(cluster.Nodes, n)
	}

	return
}

// Overlays returns overlay addresses of all nodes in the cluster
func (c *Cluster) Overlays(ctx context.Context) (o []swarm.Address, err error) {
	for _, n := range c.Nodes {
		a, err := n.Overlay(ctx)
		if err != nil {
			return []swarm.Address{}, err
		}

		o = append(o, a)
	}

	return
}

// Size returns size of the cluster
func (c *Cluster) Size() int {
	return len(c.Nodes)
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

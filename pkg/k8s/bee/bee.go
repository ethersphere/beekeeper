package bee

import (
	"github.com/ethersphere/beekeeper/pkg/k8s"
)

// Client manages communication with the Kubernetes
type Client struct {
	k8s *k8s.Client
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	KubeconfigPath string
}

// NewClient returns Kubernetes clientset
func NewClient(o *ClientOptions) (c *Client) {
	return &Client{k8s: k8s.NewClient(&k8s.ClientOptions{
		KubeconfigPath: o.KubeconfigPath,
	})}
}

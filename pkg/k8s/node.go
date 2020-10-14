package k8s

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// Node represents Bee node
type Node struct {
	k8s  *Client
	opts NodeOptions
}

// NodeOptions represents Bee node options
type NodeOptions struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
}

// NewNode returns new node
func NewNode(clientset *kubernetes.Clientset, opts NodeOptions) Node {
	return Node{
		// k8s:  clientset,
		opts: opts,
	}
}

// Create ...
func (n Node) Create(ctx context.Context, standalone bool) (err error) {
	return
}

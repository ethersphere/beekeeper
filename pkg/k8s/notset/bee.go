package notset

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/k8s"
)

// BeeClient represents not implemented Kubernetes Bee client
type BeeClient struct{}

// Create creates Bee node in the cluster
func (c *BeeClient) Create(ctx context.Context, o k8s.CreateOptions) (err error) {
	return k8s.ErrNotSet
}

// Delete deletes Bee node from the cluster
func (c *BeeClient) Delete(ctx context.Context, name string, namespace string) (err error) {
	return k8s.ErrNotSet
}

// Ready gets Bee node's readiness
func (c *BeeClient) Ready(ctx context.Context, name string, namespace string) (ready bool, err error) {
	return false, k8s.ErrNotSet
}

// RunningNodes returns list of running nodes
func (c *BeeClient) RunningNodes(ctx context.Context, namespace string) (running []string, err error) {
	return nil, k8s.ErrNotSet
}

// Start starts Bee node in the cluster
func (c *BeeClient) Start(ctx context.Context, name string, namespace string) (err error) {
	return k8s.ErrNotSet
}

// Stop stops Bee node in the cluster
func (c *BeeClient) Stop(ctx context.Context, name string, namespace string) (err error) {
	return k8s.ErrNotSet
}

// StoppedNodes returns list of stopped nodes
func (c *BeeClient) StoppedNodes(ctx context.Context, namespace string) (stopped []string, err error) {
	return nil, k8s.ErrNotSet
}

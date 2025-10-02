package restart

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/node"
)

type Client struct {
	nodeProvider node.NodeProvider
	k8sClient    *k8s.Client
	logger       logging.Logger
}

// NewClient creates a new restart client
func NewClient(nodeProvider node.NodeProvider, k8sClient *k8s.Client, l logging.Logger) *Client {
	return &Client{
		nodeProvider: nodeProvider,
		k8sClient:    k8sClient,
		logger:       l,
	}
}

func (c *Client) Restart(ctx context.Context, image string) error {
	nodes, err := c.nodeProvider.GetNodes(ctx)
	if err != nil {
		return fmt.Errorf("getting nodes: %w", err)
	}

	c.logger.Debugf("starting pod restart for %d nodes", len(nodes))
	count := 0
	for _, node := range nodes {
		if err := c.updateOrDeleteNode(ctx, node.Name(), c.nodeProvider.Namespace(), image); err != nil {
			c.logger.Warningf("failed to restart node %s: %v", node.Name(), err)
			continue
		}
		count++
	}

	c.logger.Infof("restarted %d pods", count)
	return nil
}

func (c *Client) updateOrDeleteNode(ctx context.Context, node, ns, image string) error {
	if image == "" {
		return c.deletePod(ctx, node, ns)
	}

	strategy, err := c.k8sClient.StatefulSet.GetUpdateStrategy(ctx, node, ns)
	if err != nil {
		return fmt.Errorf("getting update strategy for node %s: %w", node, err)
	}

	if err = c.k8sClient.StatefulSet.UpdateImage(ctx, node, ns, image); err != nil {
		return fmt.Errorf("updating image for node %s: %w", node, err)
	}

	if strategy.Type == statefulset.UpdateStrategyOnDelete {
		return c.deletePod(ctx, node, ns)
	}

	return nil
}

func (c *Client) deletePod(ctx context.Context, podName, ns string) error {
	ok, err := c.k8sClient.Pods.Delete(ctx, podName, ns)
	if err != nil {
		return fmt.Errorf("deleting pod %s: %w", podName, err)
	}
	if !ok {
		return fmt.Errorf("failed to delete pod %s", podName)
	}
	return nil
}

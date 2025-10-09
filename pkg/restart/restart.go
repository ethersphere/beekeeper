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

	if image == "" {
		for _, node := range nodes {
			if err := c.deletePod(ctx, node.Name(), c.nodeProvider.Namespace()); err != nil {
				return fmt.Errorf("deleting pod %s: %w", node.Name(), err)
			}
		}
		return nil
	}

	statefulSetMap := make(map[string][]string)
	for _, node := range nodes {

		pod, err := c.k8sClient.Pods.Get(ctx, node.Name(), c.nodeProvider.Namespace())
		if err != nil {
			return fmt.Errorf("getting pod %s: %w", node.Name(), err)
		}

		for _, owner := range pod.GetOwnerReferences() {
			if owner.Kind == "StatefulSet" {
				statefulSetMap[owner.Name] = append(statefulSetMap[owner.Name], node.Name())
				break
			}
		}
	}

	for ssName := range statefulSetMap {
		strategy, err := c.k8sClient.StatefulSet.GetUpdateStrategy(ctx, ssName, c.nodeProvider.Namespace())
		if err != nil {
			return fmt.Errorf("getting update strategy for statefulset %s: %w", ssName, err)
		}

		if err = c.k8sClient.StatefulSet.UpdateImage(ctx, ssName, c.nodeProvider.Namespace(), image); err != nil {
			return fmt.Errorf("updating image for statefulset %s: %w", ssName, err)
		}

		if strategy.Type == statefulset.UpdateStrategyOnDelete {
			for _, podName := range statefulSetMap[ssName] {
				if err = c.deletePod(ctx, podName, c.nodeProvider.Namespace()); err != nil {
					return fmt.Errorf("deleting pod %s: %w", podName, err)
				}
			}
		}
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

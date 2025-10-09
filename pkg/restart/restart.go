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
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	c.logger.Debugf("starting pod restart for %d nodes", len(nodes))

	if image == "" {
		return c.restartByDeletingPods(ctx, nodes)
	}

	return c.restartWithImageUpdate(ctx, nodes, image)
}

// restartByDeletingPods performs a simple restart by deleting all pods.
func (c *Client) restartByDeletingPods(ctx context.Context, nodes []node.Node) error {
	c.logger.Debug("performing simple pod deletion restart")
	podsDeleted := 0
	defer c.logger.Debugf("finished simple pod deletion restart: %d pods deleted", podsDeleted)

	for _, node := range nodes {
		if err := c.deletePod(ctx, node.Name()); err != nil {
			return fmt.Errorf("failed to delete pod %s: %w", node.Name(), err)
		}
		podsDeleted++
	}

	c.logger.Debugf("successfully deleted %d pods", podsDeleted)

	return nil
}

func (c *Client) restartWithImageUpdate(ctx context.Context, nodes node.NodeList, image string) error {
	c.logger.Debug("performing image update with pod restart")
	podsDeleted, ssUpdated := 0, 0
	defer c.logger.Debugf("finished image update with pod restart: %d pods deleted, %d statefulsets updated", podsDeleted, ssUpdated)

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

		ssUpdated++
		if strategy.Type == statefulset.UpdateStrategyOnDelete {
			for _, podName := range statefulSetMap[ssName] {
				if err = c.deletePod(ctx, podName); err != nil {
					return fmt.Errorf("deleting pod %s: %w", podName, err)
				}
				podsDeleted++
			}
		}
	}

	c.logger.Debugf("successfully updated image for %d statefulsets", ssUpdated)

	return nil
}

func (c *Client) deletePod(ctx context.Context, podName string) error {
	ok, err := c.k8sClient.Pods.Delete(ctx, podName, c.nodeProvider.Namespace())
	if err != nil {
		return fmt.Errorf("deleting pod %s: %w", podName, err)
	}
	if !ok {
		return fmt.Errorf("failed to delete pod %s", podName)
	}

	c.logger.Debugf("successfully deleted pod %s", podName)

	return nil
}

package restart

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

type Client struct {
	k8sClient *k8s.Client
	logger    logging.Logger
}

// NewClient creates a new restart client
func NewClient(k8sClient *k8s.Client, l logging.Logger) *Client {
	return &Client{
		k8sClient: k8sClient,
		logger:    l,
	}
}

func (c *Client) RestartPods(ctx context.Context, ns, labelSelector string) (err error) {
	c.logger.Infof("restarting pods in namespace %s", ns)

	if err := c.k8sClient.Pods.DeletePods(ctx, ns, labelSelector); err != nil {
		return fmt.Errorf("restarting pods in namespace %s: %w", ns, err)
	}

	return nil
}

func (c *Client) RestartCluster(ctx context.Context, cluster orchestration.Cluster, ns, image string, nodeGroups []string) (err error) {
	c.logger.Infof("restarting cluster %s", cluster.Name())

	nodes := c.getNodeList(cluster, nodeGroups)

	count := 0
	for _, node := range nodes {

		if err := c.updateOrDeleteNode(ctx, node, ns, image); err != nil {
			return fmt.Errorf("error handling node %s in namespace %s: %w", node, ns, err)
		}

		count++
		c.logger.Debugf("node %s in namespace %s deleted or updated", node, ns)
	}

	c.logger.Infof("cluster %s restarted %d/%d nodes", cluster.Name(), count, len(nodes))
	return nil
}

func (c *Client) getNodeList(cluster orchestration.Cluster, nodeGroups []string) []string {
	if len(nodeGroups) > 0 {
		var nodes []string
		for _, nodeGroup := range nodeGroups {
			if group, ok := cluster.NodeGroups()[nodeGroup]; ok {
				nodes = append(nodes, group.NodesSorted()...)
			}
		}
		return nodes
	}
	return cluster.NodeNames()
}

func (c *Client) updateOrDeleteNode(ctx context.Context, node, ns, image string) error {
	if image != "" {
		strategy, err := c.k8sClient.StatefulSet.GetUpdateStrategy(ctx, node, ns)
		if err != nil {
			return fmt.Errorf("getting update strategy for node %s: %w", node, err)
		}

		if _, err = c.k8sClient.StatefulSet.UpdateImage(ctx, node, ns, image); err != nil {
			return fmt.Errorf("updating image for node %s: %w", node, err)
		}

		if strategy.Type == statefulset.UpdateStrategyOnDelete {
			return c.deletePod(ctx, node, ns)
		}
	} else {
		return c.deletePod(ctx, node, ns)
	}
	return nil
}

func (c *Client) deletePod(ctx context.Context, node, ns string) error {
	podName := fmt.Sprintf("%s-0", node) // Suffix "-0" added as StatefulSet names pods based on replica count.
	ok, err := c.k8sClient.Pods.Delete(ctx, podName, ns)
	if err != nil {
		return fmt.Errorf("deleting pod %s: %w", podName, err)
	}
	if !ok {
		return fmt.Errorf("failed to delete pod %s", podName)
	}
	return nil
}

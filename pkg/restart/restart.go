package restart

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s"
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

func (c *Client) RestartCluster(ctx context.Context, cluster orchestration.Cluster, ns string) (err error) {
	c.logger.Infof("restarting cluster %s", cluster.Name())

	nodes := cluster.NodeNames()

	count := 0

	for _, node := range nodes {
		podName := fmt.Sprintf("%s-0", node) // Suffix "-0" added as StatefulSet names pods based on replica count.
		ok, err := c.k8sClient.Pods.Delete(ctx, podName, ns)
		if err != nil {
			return fmt.Errorf("deleting pod %s in namespace %s: %w", node, ns, err)
		}
		if ok {
			count++
			c.logger.Debugf("pod %s in namespace %s deleted", podName, ns)
		}
	}

	c.logger.Infof("cluster %s restarted %d/%d nodes", cluster.Name(), count, len(nodes))

	return nil
}

func (c *Client) RestartPods(ctx context.Context, ns, labelSelector string) (err error) {
	c.logger.Infof("restarting pods in namespace %s", ns)

	if err := c.k8sClient.Pods.DeletePods(ctx, ns, labelSelector); err != nil {
		return fmt.Errorf("restarting pods in namespace %s: %w", ns, err)
	}

	return nil
}

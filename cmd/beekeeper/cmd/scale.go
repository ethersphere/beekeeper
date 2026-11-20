package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	orchestrationK8S "github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
	"github.com/spf13/cobra"
)

func (c *command) initScaleCmd() (err error) {
	const (
		optionNameNodeGroup = "node-group"
		optionNameCount     = "count"
		optionNameTimeout   = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "scale",
		Short: "scales a node group in a running Bee cluster to the given count",
		Long: `Scales a node group in a running Bee cluster to the given node count.

The node group's nodes are named <node-group>-0, <node-group>-1, ...; the command
grows or shrinks from the last index to reach the requested count:
• if count is greater than the current size, the missing nodes are deployed
• if count is less than the current size, the highest-indexed nodes are deleted
• if count equals the current size, nothing is done

New nodes are not funded by this command; run "beekeeper node-funder" afterwards
if they need ETH/BZZ.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			clusterName := c.globalConfig.GetString(optionNameClusterName)
			ngName := c.globalConfig.GetString(optionNameNodeGroup)
			count := c.globalConfig.GetInt(optionNameCount)

			if ngName == "" {
				return fmt.Errorf("node group name not provided")
			}
			if count < 0 {
				return fmt.Errorf("count must be 0 or greater")
			}

			start := time.Now()
			err = c.scaleNodeGroup(ctx, clusterName, ngName, count)
			c.log.Infof("scale took %s", time.Since(start))
			return err
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "cluster name")
	cmd.Flags().String(optionNameNodeGroup, "", "node group to scale. Required")
	cmd.Flags().Int(optionNameCount, 0, "desired node count for the node group. Required")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

// scaleNodeGroup reconciles a node group to exactly count nodes: growing from the last index if count is larger than the current size, shrinking from the last index if smaller, or doing nothing if count already matches.
func (c *command) scaleNodeGroup(ctx context.Context, clusterName, ngName string, count int) error {
	if clusterName == "" {
		return errMissingClusterName
	}

	clusterConfig, ok := c.config.Clusters[clusterName]
	if !ok {
		return fmt.Errorf("cluster %s not defined", clusterName)
	}

	if clusterConfig.IsUsingStaticEndpoints() {
		return fmt.Errorf("static endpoints are not supported for scaling")
	}

	ngv, ok := clusterConfig.GetNodeGroups()[ngName]
	if !ok {
		return fmt.Errorf("node group %s not defined in cluster %s", ngName, clusterName)
	}
	if ngv.Mode == bootnodeMode {
		return fmt.Errorf("scaling bootnode node groups is not supported")
	}
	if len(ngv.Nodes) > 0 {
		return fmt.Errorf("node group %s uses explicit nodes, not count-based scaling", ngName)
	}

	ngConfig, ok := c.config.NodeGroups[ngv.Config]
	if !ok {
		return fmt.Errorf("node group profile %s not defined", ngv.Config)
	}

	namespace := clusterConfig.GetNamespace()

	currentSize, err := c.nodeGroupSize(ctx, ngName, namespace)
	if err != nil {
		return fmt.Errorf("determine current size of node group %s: %w", ngName, err)
	}

	switch {
	case count > currentSize:
		return c.growNodeGroup(ctx, clusterConfig, ngv, ngConfig, ngName, currentSize, count)
	case count < currentSize:
		return c.shrinkNodeGroup(ctx, clusterConfig, ngConfig, ngName, currentSize, count)
	default:
		c.log.Infof("node group %s already has %d nodes, nothing to do", ngName, count)
		return nil
	}
}

// nodeGroupSize returns the current number of nodes in the node group by probing live StatefulSets from index 0 upward until one is missing.
// Note: Cannot get from local.yaml config because the node group may have been scaled up or down since the cluster was deployed.
func (c *command) nodeGroupSize(ctx context.Context, ngName, namespace string) (int, error) {
	size := 0
	for {
		name := fmt.Sprintf("%s-%d", ngName, size)
		exists, err := c.k8sClient.StatefulSet.Exists(ctx, name, namespace)
		if err != nil {
			return 0, fmt.Errorf("checking node %s: %w", name, err)
		}
		if !exists {
			return size, nil
		}
		size++
	}
}

// growNodeGroup adds nodes to the end of the node group (lowest missing index first) until count are reached.
// For example, going from 5 to 8 deploys <ngName>-5, <ngName>-6, and <ngName>-7, leaving <ngName>-0..4 untouched.
func (c *command) growNodeGroup(
	ctx context.Context,
	clusterConfig config.Cluster,
	ngv config.ClusterNodeGroup,
	ngConfig config.NodeGroup,
	ngName string,
	currentSize, count int,
) error {
	beeConfig, ok := c.config.BeeConfigs[ngv.BeeConfig]
	if !ok {
		return fmt.Errorf("bee profile %s not defined", ngv.BeeConfig)
	}
	bConfig := beeConfig.Export()

	if bConfig.Bootnodes == nil || len(*bConfig.Bootnodes) == 0 {
		bootnodes, err := bootnodesForCluster(clusterConfig, c.config)
		if err != nil {
			return fmt.Errorf("resolve bootnodes: %w", err)
		}
		if bootnodes != "" {
			bConfig.Bootnodes = &[]string{bootnodes}
		}
	}

	ngOptions := ngConfig.Export()
	ngOptions.BeeConfig = &bConfig

	newNames := make([]string, 0, count-currentSize)
	for i := currentSize; i < count; i++ {
		newNames = append(newNames, fmt.Sprintf("%s-%d", ngName, i))
	}

	c.log.Infof("scaling node group %s from %d to %d: deploying %v", ngName, currentSize, count, newNames)

	cluster := orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterConfig.Export(), c.k8sClient, c.swapClient, c.log)
	cluster.AddNodeGroup(ngName, ngOptions)

	ng, err := cluster.NodeGroup(ngName)
	if err != nil {
		return fmt.Errorf("get node group: %w", err)
	}

	inCluster := c.globalConfig.GetBool(optionNameInCluster)

	errChan := make(chan error)
	defer close(errChan)

	for _, name := range newNames {
		go func(name string) {
			_, err := ng.DeployNode(ctx, name, inCluster, orchestration.NodeOptions{})
			errChan <- err
		}(name)
	}

	for range newNames {
		if err := <-errChan; err != nil {
			return fmt.Errorf("deploy node: %w", err)
		}
	}

	c.log.Infof("node group %s scaled to %d nodes", ngName, count)

	return nil
}

// shrinkNodeGroup removes nodes from the end of the node group (highest index first) until only count remain.
// For example, going from 8 to 5 deletes <ngName>-7, <ngName>-6, and <ngName>-5, leaving <ngName>-0..4 untouched.
func (c *command) shrinkNodeGroup(
	ctx context.Context,
	clusterConfig config.Cluster,
	ngConfig config.NodeGroup,
	ngName string,
	currentSize, count int,
) error {
	namespace := clusterConfig.GetNamespace()

	cluster := orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterConfig.Export(), c.k8sClient, c.swapClient, c.log)
	cluster.AddNodeGroup(ngName, ngConfig.Export())

	ng, err := cluster.NodeGroup(ngName)
	if err != nil {
		return fmt.Errorf("get node group: %w", err)
	}

	c.log.Infof("node group %s: shrinking from %d to %d nodes", ngName, currentSize, count)

	// delete from the highest index down, so the remaining nodes are always a contiguous 0..count-1 range with no gaps
	for i := currentSize - 1; i >= count; i-- {
		name := fmt.Sprintf("%s-%d", ngName, i)

		c.log.Infof("deleting node %s", name)
		if err := ng.DeleteNode(ctx, name); err != nil {
			return fmt.Errorf("deleting node %s: %w", name, err)
		}

		// the StatefulSet's PVC outlives the StatefulSet itself, so it needs an explicit delete when persistence is enabled for this node group
		if ngConfig.PersistenceEnabled != nil && *ngConfig.PersistenceEnabled {
			pvcName := fmt.Sprintf("data-%s-0", name)
			if err := c.k8sClient.PVC.Delete(ctx, pvcName, namespace); err != nil {
				return fmt.Errorf("deleting pvc %s: %w", pvcName, err)
			}
		}
	}

	c.log.Infof("node group %s: now has %d nodes", ngName, count)

	return nil
}

// bootnodesForCluster resolves the bootnode multiaddr for a cluster the same way cluster setup
func bootnodesForCluster(clusterConfig config.Cluster, cfg *config.Config) (string, error) {
	for _, v := range clusterConfig.GetNodeGroups() {
		if v.Mode != bootnodeMode {
			continue
		}
		for _, node := range v.Nodes {
			if node.Bootnodes == "" {
				continue
			}
			return fmt.Sprintf(node.Bootnodes, clusterConfig.GetNamespace()), nil
		}
	}
	return "", nil
}

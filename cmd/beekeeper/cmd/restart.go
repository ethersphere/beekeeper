package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initRestartCmd() (err error) {
	const (
		optionNameClusterName   = "cluster-name"
		optionNameLabelSelector = "label-selector"
		optionNameNamespace     = "namespace"
		optionNameTimeout       = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart pods in a cluster or namespace",
		Long:  `Restarts pods by deleting them. Uses cluster name as the primary scope or falls back to namespace, with optional label filtering.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			clusterName := c.globalConfig.GetString(optionNameClusterName)
			namespace := c.globalConfig.GetString(optionNameNamespace)

			if clusterName == "" && namespace == "" {
				return errors.New("either cluster name or namespace must be provided")
			}

			if clusterName != "" {
				if err := c.restartCluster(ctx, clusterName, c.config); err != nil {
					return fmt.Errorf("restarting cluster %s: %w", clusterName, err)
				}
				return nil
			}

			if err := c.k8sClient.Pods.DeletePods(ctx, namespace, c.globalConfig.GetString(optionNameLabelSelector)); err != nil {
				return fmt.Errorf("restarting pods in namespace %s: %w", namespace, err)
			}

			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "Kubernetes cluster to operate on (overrides namespace).")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Namespace to delete pods from (used if cluster name is not set).")
	cmd.Flags().String(optionNameLabelSelector, "", "Label selector for resources in the namespace. Ignored if cluster name is set.")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Operation timeout (e.g., 5s, 10m, 1.5h).")

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) restartCluster(ctx context.Context, clusterName string, cfg *config.Config) (err error) {
	c.log.Infof("restarting cluster %s", clusterName)

	clusterConfig, ok := cfg.Clusters[clusterName]
	if !ok {
		return fmt.Errorf("cluster config %s not defined", clusterName)
	}

	cluster, err := c.setupCluster(ctx, clusterName, c.config, false)
	if err != nil {
		return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
	}

	nodes := cluster.NodeNames()

	count := 0

	for _, node := range nodes {
		podName := fmt.Sprintf("%s-0", node) // Suffix "-0" added as StatefulSet names pods based on replica count.
		ok, err := c.k8sClient.Pods.Delete(ctx, podName, clusterConfig.GetNamespace())
		if err != nil {
			return fmt.Errorf("deleting pod %s in namespace %s: %w", node, clusterConfig.GetNamespace(), err)
		}
		if ok {
			count++
			c.log.Debugf("pod %s in namespace %s deleted", podName, clusterConfig.GetNamespace())
		}
	}

	c.log.Infof("cluster %s restarted %d/%d nodes", clusterName, count, len(nodes))

	return nil
}

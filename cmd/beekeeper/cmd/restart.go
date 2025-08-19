package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/restart"
	"github.com/spf13/cobra"
)

func (c *command) initRestartCmd() (err error) {
	const (
		optionNameLabelSelector = "label-selector"
		optionNameNamespace     = "namespace"
		optionNameImage         = "image"
		optionNameNodeGroups    = "node-groups"
		optionNameTimeout       = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restart pods in a cluster or namespace",
		Long: `Restarts Bee nodes in a Kubernetes cluster or namespace by deleting and recreating pods.

The restart command provides two restart strategies:
• Cluster-based restart: Restart all nodes in a Beekeeper-managed cluster
• Namespace-based restart: Restart pods in a specific namespace with optional filtering

Restart options:
• Use --image to specify a new container image for the restarted pods
• Use --node-groups to target specific node groups within a cluster
• Use --label-selector to filter which pods to restart in a namespace

This command is useful for:
• Applying configuration changes that require pod restart
• Updating to new Bee versions
• Troubleshooting node issues
• Rolling out updates across the cluster

Requires either --cluster-name or --namespace to be specified.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			clusterName := c.globalConfig.GetString(optionNameClusterName)
			namespace := c.globalConfig.GetString(optionNameNamespace)

			if clusterName == "" && namespace == "" {
				return errors.New("either cluster name or namespace must be provided")
			}

			restartClient := restart.NewClient(c.k8sClient, c.log)

			if clusterName != "" {
				clusterConfig, ok := c.config.Clusters[clusterName]
				if !ok {
					return fmt.Errorf("cluster config %s not defined", clusterName)
				}

				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
				}

				c.log.Infof("restarting cluster %s", clusterName)

				if err := restartClient.RestartCluster(ctx,
					cluster,
					clusterConfig.GetNamespace(),
					c.globalConfig.GetString(optionNameImage),
					c.globalConfig.GetStringSlice(optionNameNodeGroups),
				); err != nil {
					return fmt.Errorf("restarting cluster %s: %w", clusterName, err)
				}

				return nil
			}

			if err := restartClient.RestartPods(ctx, namespace, c.globalConfig.GetString(optionNameLabelSelector)); err != nil {
				return fmt.Errorf("restarting pods in namespace %s: %w", namespace, err)
			}

			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "Kubernetes cluster to operate on (overrides namespace and label selector).")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Namespace to delete pods from (only used if cluster name is not set).")
	cmd.Flags().String(optionNameLabelSelector, "", "Label selector for resources in the namespace (only used with namespace).")
	cmd.Flags().String(optionNameImage, "", "Container image to use when restarting pods (defaults to current image if not set).")
	cmd.Flags().StringSlice(optionNameNodeGroups, nil, "List of node groups to target for restarts (applies to all groups if not set).")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Operation timeout (e.g., 5s, 10m, 1.5h).")

	c.root.AddCommand(cmd)

	return nil
}

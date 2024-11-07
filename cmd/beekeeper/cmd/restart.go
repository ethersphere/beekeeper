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

			restartClient := restart.NewClient(c.k8sClient, c.log)

			if clusterName != "" {
				clusterConfig, ok := c.config.Clusters[clusterName]
				if !ok {
					return fmt.Errorf("cluster config %s not defined", clusterName)
				}

				cluster, err := c.setupCluster(ctx, clusterName, c.config, false)
				if err != nil {
					return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
				}

				if err := restartClient.RestartCluster(ctx, cluster, clusterConfig.GetNamespace()); err != nil {
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

	cmd.Flags().String(optionNameClusterName, "", "Kubernetes cluster to operate on (overrides namespace).")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Namespace to delete pods from (used if cluster name is not set).")
	cmd.Flags().String(optionNameLabelSelector, "", "Label selector for resources in the namespace. Ignored if cluster name is set.")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Operation timeout (e.g., 5s, 10m, 1.5h).")

	c.root.AddCommand(cmd)

	return nil
}

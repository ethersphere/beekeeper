package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/restart"
	"github.com/spf13/cobra"
)

func (c *command) initRestartCmd() (err error) {
	const (
		optionNameLabelSelector  = "label-selector"
		optionNameNamespace      = "namespace"
		optionNameImage          = "image"
		optionNameNodeGroups     = "node-groups"
		optionNameTimeout        = "timeout"
		optionNameDeploymentType = "deployment-type"
	)

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart pods in a cluster or namespace",
		Long:  `Restarts pods by deleting them. Uses cluster name as the primary scope or falls back to namespace, with optional label filtering.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {

				nodeClient, err := c.createNodeClient(ctx, true)
				if err != nil {
					return fmt.Errorf("creating node client: %w", err)
				}

				restartClient := restart.NewClient(nodeClient, c.k8sClient, c.log)

				// TODO: Add cluster restart (should be handled by the node client)
				if err := restartClient.RestartPods(ctx); err != nil {
					return fmt.Errorf("restarting pods: %w", err)
				}

				return nil
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "Kubernetes cluster to operate on (overrides namespace and label selector).")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Namespace to delete pods from (only used if cluster name is not set).")
	cmd.Flags().String(optionNameLabelSelector, "", "Label selector for resources in the namespace (only used with namespace).")
	cmd.Flags().String(optionNameImage, "", "Container image to use when restarting pods (defaults to current image if not set).")
	cmd.Flags().StringSlice(optionNameNodeGroups, nil, "List of node groups to target for restarts (applies to all groups if not set).")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Operation timeout (e.g., 5s, 10m, 1.5h).")
	cmd.Flags().String(optionNameDeploymentType, "beekeeper", "Indicates how the cluster was deployed: 'beekeeper' or 'helm'.")

	c.root.AddCommand(cmd)

	return nil
}

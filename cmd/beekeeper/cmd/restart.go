package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func (c *command) initRestartCmd() (err error) {
	const (
		optionNameNamespace     = "namespace"
		optionNameTimeout       = "timeout"
		optionNameLabelSelector = "label-selector"
	)

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restarts pods in a namespace",
		Long:  `Restarts pods in a namespace by deleting them and using labels to filter resources.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var namespace string
			if namespace = c.globalConfig.GetString(optionNameNamespace); namespace == "" {
				return errors.New("namespace not provided")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			if err := c.k8sClient.Pods.DeletePods(ctx, namespace, c.globalConfig.GetString(optionNameLabelSelector)); err != nil {
				return fmt.Errorf("restarting pods in namespace %s: %w", namespace, err)
			}

			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace to delete pods from. Required.")
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the namespace. An empty string selects all resources.")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Timeout. Example: 5s, 10m, 1.5h")

	c.root.AddCommand(cmd)

	return nil
}

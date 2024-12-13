package cmd

import (
	"time"

	"github.com/ethersphere/beekeeper/pkg/stamper"
	"github.com/spf13/cobra"
)

func (c *command) initStamperCmd() (err error) {
	const (
		optionNameNamespace     = "namespace"
		optionTTLTreshold       = "ttl-treshold"
		optionTopUpTo           = "topup-to"
		optionUsageThreshold    = "usage-threshold"
		optionDiutionDepth      = "dilution-depth"
		optionPeriodicCheck     = "periodic-check"
		optionNameTimeout       = "timeout"
		optionNameLabelSelector = "label-selector"
	)

	cmd := &cobra.Command{
		Use:   "stamper",
		Short: "Manage postage batches for nodes",
		Long:  `Use the stamper command to manage postage batches for nodes. Topup, dilution and creation of postage batches are supported.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			namespace := c.globalConfig.GetString(optionNameNamespace)
			// clusterName := c.globalConfig.GetString(optionNameClusterName)

			stamper := stamper.NewClient(&stamper.ClientConfig{
				Log:           c.log,
				Namespace:     namespace,
				K8sClient:     c.k8sClient,
				LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
			})

			_ = stamper

			return
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace. Overrides cluster name if set.")
	cmd.Flags().String(optionNameClusterName, "", "Name of the Beekeeper cluster to target. Ignored if a namespace is specified, in which case the namespace from the cluster configuration is used.")
	cmd.Flags().Duration(optionTTLTreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().Float64(optionUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint16(optionDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the specified namespace. Use an empty string to select all resources.")
	cmd.Flags().Duration(optionNameTimeout, 0*time.Minute, "Maximum duration to wait for the operation to complete. Default is no timeout.")

	c.root.AddCommand(cmd)

	return nil
}

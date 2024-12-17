package cmd

import (
	"time"

	"github.com/ethersphere/beekeeper/pkg/stamper"
	"github.com/spf13/cobra"
)

func (c *command) initStamperCmd() (err error) {
	const (
		optionNameNamespace     = "namespace"
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

			c.stamper = stamper.NewStamperClient(&stamper.ClientConfig{
				Log:           c.log,
				Namespace:     namespace,
				K8sClient:     c.k8sClient,
				LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
				InCluster:     c.globalConfig.GetBool(optionNameInCluster),
			})

			return
		},
		PreRunE: c.preRunE,
	}

	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace. Overrides cluster name if set.")
	cmd.PersistentFlags().String(optionNameClusterName, "", "Name of the Beekeeper cluster to target. Ignored if a namespace is specified, in which case the namespace from the cluster configuration is used.")
	cmd.PersistentFlags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the specified namespace. Use an empty string to select all resources.")
	cmd.PersistentFlags().Duration(optionNameTimeout, 0*time.Minute, "Maximum duration to wait for the operation to complete. Default is no timeout.")

	cmd.AddCommand(c.initStamperTopup())
	cmd.AddCommand(c.initStamperDilute())
	cmd.AddCommand(c.initStamperCreate())
	cmd.AddCommand(c.initStamperSet())

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) initStamperTopup() *cobra.Command {
	const (
		optionTTLThreshold = "ttl-threshold"
		optionTopUpTo      = "topup-to"
	)

	cmd := &cobra.Command{
		Use:   "topup",
		Short: "Top up the TTL of postage batches",
		Long:  `Top up the TTL of postage batches.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return
		},
	}

	cmd.Flags().Duration(optionTTLThreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")

	c.root.AddCommand(cmd)

	return cmd
}

func (c *command) initStamperDilute() *cobra.Command {
	const (
		optionUsageThreshold = "usage-threshold"
		optionDiutionDepth   = "dilution-depth"
	)

	cmd := &cobra.Command{
		Use:   "dilute",
		Short: "Dilute postage batches",
		Long:  `Dilute postage batches.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return
		},
	}

	cmd.Flags().Float64(optionUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint16(optionDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")

	c.root.AddCommand(cmd)

	return cmd
}

func (c *command) initStamperCreate() *cobra.Command {
	const (
		optionNameAmount = "amount"
		optionNameDepth  = "depth"
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new postage batch",
		Long:  `Create a new postage batch.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return
		},
	}

	cmd.Flags().Uint64(optionNameAmount, 1000000000, "Amount of tokens to be staked in the postage batch.")
	cmd.Flags().Uint8(optionNameDepth, 8, "Depth of the postage batch.")

	c.root.AddCommand(cmd)

	return cmd
}

func (c *command) initStamperSet() *cobra.Command {
	const (
		optionTTLThreshold   = "ttl-threshold"
		optionTopUpTo        = "topup-to"
		optionUsageThreshold = "usage-threshold"
		optionDiutionDepth   = "dilution-depth"
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set stamper configuration",
		Long:  `Set stamper configuration.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return
		},
	}

	cmd.Flags().Duration(optionTTLThreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().Float64(optionUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint16(optionDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")

	c.root.AddCommand(cmd)

	return cmd
}

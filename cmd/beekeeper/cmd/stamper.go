package cmd

import (
	"context"
	"time"

	"github.com/ethersphere/beekeeper/pkg/scheduler"
	"github.com/ethersphere/beekeeper/pkg/stamper"
	"github.com/spf13/cobra"
)

const (
	optionNamePeriodicCheck string = "periodic-check"
	optionNameNamespace     string = "namespace"
	optionNameLabelSelector string = "label-selector"
)

func (c *command) initStamperCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "stamper",
		Short: "Manage postage batches for nodes",
		Long:  `Use the stamper command to manage postage batches for nodes. Topup, dilution and creation of postage batches are supported.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
	}

	cmd.AddCommand(initStamperDefaultFlags(c.initStamperTopup()))
	cmd.AddCommand(initStamperDefaultFlags(c.initStamperDilute()))
	cmd.AddCommand(initStamperDefaultFlags(c.initStamperCreate()))
	cmd.AddCommand(initStamperDefaultFlags(c.initStamperSet()))

	c.root.AddCommand(cmd)

	return nil
}

func initStamperDefaultFlags(cmd *cobra.Command) *cobra.Command {
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace (overrides cluster name).")
	cmd.PersistentFlags().String(optionNameClusterName, "", "Target Beekeeper cluster name.")
	cmd.PersistentFlags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources (use empty string for all).")
	cmd.PersistentFlags().Duration(optionNameTimeout, 0*time.Minute, "Operation timeout (no timeout by default).")
	cmd.PersistentFlags().Duration(optionNamePeriodicCheck, 0*time.Minute, "Periodic stamper check interval (none by default).")
	return cmd
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
			var ctx context.Context
			var cancel context.CancelFunc
			timeout := c.globalConfig.GetDuration(optionNameTimeout)
			if timeout > 0 {
				ctx, cancel = context.WithTimeout(cmd.Context(), timeout)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			namespace := c.globalConfig.GetString(optionNameNamespace)
			// clusterName := c.globalConfig.GetString(optionNameClusterName)

			c.stamper = stamper.NewStamperClient(&stamper.ClientConfig{
				Log:           c.log,
				Namespace:     namespace,
				K8sClient:     c.k8sClient,
				LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
				InCluster:     c.globalConfig.GetBool(optionNameInCluster),
			})

			periodicCheck := c.globalConfig.GetDuration(optionNamePeriodicCheck)

			if periodicCheck == 0 {
				return c.stamper.Dilute(ctx, c.globalConfig.GetFloat64(optionUsageThreshold), c.globalConfig.GetUint16(optionDiutionDepth))
			}

			diluteExecutor := scheduler.NewPeriodicExecutor(periodicCheck, c.log)
			diluteExecutor.Start(ctx, func(ctx context.Context) error {
				return c.stamper.Dilute(ctx, c.globalConfig.GetFloat64(optionUsageThreshold), c.globalConfig.GetUint16(optionDiutionDepth))
			})
			defer diluteExecutor.Stop()

			<-ctx.Done()

			c.log.Infof("dilution stopped: %v", ctx.Err())

			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Float64(optionUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint8(optionDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")

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
package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
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
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace (overrides cluster name).")
	cmd.Flags().String(optionNameClusterName, "", "Target Beekeeper cluster name.")
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources (use empty string for all).")
	cmd.Flags().Duration(optionNameTimeout, 0*time.Minute, "Operation timeout.")
	return cmd
}

func (c *command) initStamperTopup() *cobra.Command {
	const (
		optionTTLThreshold = "ttl-threshold"
		optionTopUpTo      = "topup-to"
		optionGethUrl      = "geth-url"
	)

	cmd := &cobra.Command{
		Use:   "topup",
		Short: "Top up the TTL of postage batches",
		Long:  `Top up the TTL of postage batches.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			timeout := c.globalConfig.GetDuration(optionNameTimeout)
			ctx := cmd.Context()
			var cancel context.CancelFunc

			if timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			namespace := c.globalConfig.GetString(optionNameNamespace)
			clusterName := c.globalConfig.GetString(optionNameClusterName)

			if clusterName == "" && namespace == "" {
				return errors.New("either cluster name or namespace must be provided")
			}

			var beeClients map[string]*bee.Client

			if clusterName != "" {
				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
				}

				beeClients, err = cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to retrieve node clients: %w", err)
				}
			}

			c.stamper = stamper.NewStamperClient(&stamper.ClientConfig{
				Log:           c.log,
				Namespace:     namespace,
				K8sClient:     c.k8sClient,
				SwapClient:    c.swapClient,
				BeeClients:    beeClients,
				LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
				InCluster:     c.globalConfig.GetBool(optionNameInCluster),
			})

			return c.executePeriodically(ctx, func(ctx context.Context) error {
				return c.stamper.Topup(ctx, c.globalConfig.GetDuration(optionTTLThreshold), c.globalConfig.GetDuration(optionTopUpTo))
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Duration(optionTTLThreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().String(optionGethUrl, "", "Geth URL for chain state retrieval.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

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
			timeout := c.globalConfig.GetDuration(optionNameTimeout)
			ctx := cmd.Context()
			var cancel context.CancelFunc

			if timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			namespace := c.globalConfig.GetString(optionNameNamespace)
			clusterName := c.globalConfig.GetString(optionNameClusterName)

			if clusterName == "" && namespace == "" {
				return errors.New("either cluster name or namespace must be provided")
			}

			var beeClients map[string]*bee.Client

			if clusterName != "" {
				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
				}

				beeClients, err = cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to retrieve node clients: %w", err)
				}
			}

			c.stamper = stamper.NewStamperClient(&stamper.ClientConfig{
				Log:           c.log,
				Namespace:     namespace,
				K8sClient:     c.k8sClient,
				BeeClients:    beeClients,
				LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
				InCluster:     c.globalConfig.GetBool(optionNameInCluster),
			})

			return c.executePeriodically(ctx, func(ctx context.Context) error {
				return c.stamper.Dilute(ctx, c.globalConfig.GetFloat64(optionUsageThreshold), c.globalConfig.GetUint16(optionDiutionDepth))
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Float64(optionUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint8(optionDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

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
		Short: "Create a postage batch for selected nodes",
		Long:  `Create a postage batch for selected nodes. Nodes are selected by namespace (use label-selector for filtering) or cluster name.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			timeout := c.globalConfig.GetDuration(optionNameTimeout)
			ctx := cmd.Context()
			var cancel context.CancelFunc

			if timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			namespace := c.globalConfig.GetString(optionNameNamespace)
			clusterName := c.globalConfig.GetString(optionNameClusterName)

			if clusterName == "" && namespace == "" {
				return errors.New("either cluster name or namespace must be provided")
			}

			var beeClients map[string]*bee.Client

			if clusterName != "" {
				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
				}

				beeClients, err = cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to retrieve node clients: %w", err)
				}
			}

			c.stamper = stamper.NewStamperClient(&stamper.ClientConfig{
				Log:           c.log,
				Namespace:     namespace,
				K8sClient:     c.k8sClient,
				BeeClients:    beeClients,
				LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
				InCluster:     c.globalConfig.GetBool(optionNameInCluster),
			})

			amount := c.globalConfig.GetUint64(optionNameAmount)
			depth := c.globalConfig.GetUint16(optionNameDepth)

			return c.stamper.Create(ctx, amount, depth)
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Uint64(optionNameAmount, 100000000, "Amount of BZZ in PLURS added that the postage batch will have.")
	cmd.Flags().Uint16(optionNameDepth, 16, "Batch depth which specifies how many chunks can be signed with the batch. It is a logarithm. Must be higher than default bucket depth (16)")

	c.root.AddCommand(cmd)

	return cmd
}

func (c *command) initStamperSet() *cobra.Command {
	const (
		optionTTLThreshold   = "ttl-threshold"
		optionTopUpTo        = "topup-to"
		optionUsageThreshold = "usage-threshold"
		optionDiutionDepth   = "dilution-depth"
		optionGethUrl        = "geth-url"
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set stamper configuration",
		Long:  `Set stamper configuration.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			timeout := c.globalConfig.GetDuration(optionNameTimeout)
			ctx := cmd.Context()
			var cancel context.CancelFunc

			if timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			namespace := c.globalConfig.GetString(optionNameNamespace)
			clusterName := c.globalConfig.GetString(optionNameClusterName)

			if clusterName == "" && namespace == "" {
				return errors.New("either cluster name or namespace must be provided")
			}

			var beeClients map[string]*bee.Client

			if clusterName != "" {
				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
				}

				beeClients, err = cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to retrieve node clients: %w", err)
				}
			}

			c.stamper = stamper.NewStamperClient(&stamper.ClientConfig{
				Log:           c.log,
				Namespace:     namespace,
				K8sClient:     c.k8sClient,
				SwapClient:    c.swapClient,
				BeeClients:    beeClients,
				LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
				InCluster:     c.globalConfig.GetBool(optionNameInCluster),
			})

			return c.executePeriodically(ctx, func(ctx context.Context) error {
				return c.stamper.Set(ctx,
					c.globalConfig.GetDuration(optionTTLThreshold),
					c.globalConfig.GetDuration(optionTopUpTo),
					c.globalConfig.GetFloat64(optionUsageThreshold),
					c.globalConfig.GetUint16(optionDiutionDepth),
				)
			})
		},
	}

	cmd.Flags().Duration(optionTTLThreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().Float64(optionUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint16(optionDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")
	cmd.Flags().String(optionGethUrl, "", "Geth URL for chain state retrieval.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

	c.root.AddCommand(cmd)

	return cmd
}

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
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Operation timeout (e.g., 5s, 10m, 1.5h).")
	return cmd
}

func (c *command) initStamperTopup() *cobra.Command {
	const (
		optionNameTTLThreshold = "ttl-threshold"
		optionNameTopUpTo      = "topup-to"
		optionNameGethUrl      = "geth-url"
		optionNameBatchIDs     = "batch-ids"
	)

	cmd := &cobra.Command{
		Use:   "topup",
		Short: "Top up the TTL of postage batches",
		Long:  `Top up the TTL of postage batches.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				stamperClient, err := c.createStamperClient(ctx)
				if err != nil {
					return fmt.Errorf("failed to create stamper client: %w", err)
				}

				return c.executePeriodically(ctx, func(ctx context.Context) error {
					return stamperClient.Topup(ctx,
						c.globalConfig.GetDuration(optionNameTTLThreshold),
						c.globalConfig.GetDuration(optionNameTopUpTo),
						stamper.WithBatchIDs(c.globalConfig.GetStringSlice(optionNameBatchIDs)),
					)
				})
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Duration(optionNameTTLThreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionNameTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().StringSlice(optionNameBatchIDs, nil, "Comma separated list of postage batch IDs to top up. If not provided, all batches are topped up.")
	cmd.Flags().String(optionNameGethUrl, "", "Geth URL for chain state retrieval.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

	return cmd
}

func (c *command) initStamperDilute() *cobra.Command {
	const (
		optionNameUsageThreshold = "usage-threshold"
		optionNameDiutionDepth   = "dilution-depth"
		optionNameBatchIDs       = "batch-ids"
	)

	cmd := &cobra.Command{
		Use:   "dilute",
		Short: "Dilute postage batches",
		Long:  `Dilute postage batches.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				stamperClient, err := c.createStamperClient(ctx)
				if err != nil {
					return fmt.Errorf("failed to create stamper client: %w", err)
				}

				return c.executePeriodically(ctx, func(ctx context.Context) error {
					return stamperClient.Dilute(ctx,
						c.globalConfig.GetFloat64(optionNameUsageThreshold),
						c.globalConfig.GetUint16(optionNameDiutionDepth),
						stamper.WithBatchIDs(c.globalConfig.GetStringSlice(optionNameBatchIDs)),
					)
				})
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Float64(optionNameUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint8(optionNameDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")
	cmd.Flags().StringSlice(optionNameBatchIDs, nil, "Comma separated list of postage batch IDs to dilute. If not provided, all batches are diluted.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

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
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				stamperClient, err := c.createStamperClient(ctx)
				if err != nil {
					return fmt.Errorf("failed to create stamper client: %w", err)
				}

				return stamperClient.Create(ctx,
					c.globalConfig.GetUint64(optionNameAmount),
					c.globalConfig.GetUint16(optionNameDepth),
				)
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Uint64(optionNameAmount, 100000000, "Amount of BZZ in PLURS added that the postage batch will have.")
	cmd.Flags().Uint16(optionNameDepth, 17, "Batch depth which specifies how many chunks can be signed with the batch. It is a logarithm. Must be higher than default bucket depth (16)")

	return cmd
}

func (c *command) initStamperSet() *cobra.Command {
	const (
		optionNameTTLThreshold   = "ttl-threshold"
		optionNameTopUpTo        = "topup-to"
		optionNameUsageThreshold = "usage-threshold"
		optionNameDiutionDepth   = "dilution-depth"
		optionNameGethUrl        = "geth-url"
		optionNameBatchIDs       = "batch-ids"
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set stamper configuration",
		Long:  `Set stamper configuration.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				stamperClient, err := c.createStamperClient(ctx)
				if err != nil {
					return fmt.Errorf("failed to create stamper client: %w", err)
				}

				return c.executePeriodically(ctx, func(ctx context.Context) error {
					return stamperClient.Set(ctx,
						c.globalConfig.GetDuration(optionNameTTLThreshold),
						c.globalConfig.GetDuration(optionNameTopUpTo),
						c.globalConfig.GetFloat64(optionNameUsageThreshold),
						c.globalConfig.GetUint16(optionNameDiutionDepth),
						stamper.WithBatchIDs(c.globalConfig.GetStringSlice(optionNameBatchIDs)),
					)
				})
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Duration(optionNameTTLThreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionNameTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().Float64(optionNameUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint16(optionNameDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")
	cmd.Flags().StringSlice(optionNameBatchIDs, nil, "Comma separated list of postage batch IDs to set. If not provided, all batches are set.")
	cmd.Flags().String(optionNameGethUrl, "", "Geth URL for chain state retrieval.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

	return cmd
}

func (c *command) createStamperClient(ctx context.Context) (*stamper.Client, error) {
	namespace := c.globalConfig.GetString(optionNameNamespace)
	clusterName := c.globalConfig.GetString(optionNameClusterName)

	if clusterName == "" && namespace == "" {
		return nil, errors.New("either cluster name or namespace must be provided")
	}

	var beeClients map[string]*bee.Client

	if clusterName != "" {
		cluster, err := c.setupCluster(ctx, clusterName, false)
		if err != nil {
			return nil, fmt.Errorf("setting up cluster %s: %w", clusterName, err)
		}

		beeClients, err = cluster.NodesClients(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve node clients: %w", err)
		}
	}

	return stamper.New(&stamper.ClientConfig{
		Log:           c.log,
		Namespace:     namespace,
		K8sClient:     c.k8sClient,
		BeeClients:    beeClients,
		SwapClient:    c.swapClient,
		LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
		InCluster:     c.globalConfig.GetBool(optionNameInCluster),
	}), nil
}

func (c *command) withTimeoutHandler(cmd *cobra.Command, f func(ctx context.Context) error) error {
	timeout := c.globalConfig.GetDuration(optionNameTimeout)
	ctx := cmd.Context()
	var cancel context.CancelFunc

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return f(ctx)
}

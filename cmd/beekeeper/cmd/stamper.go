package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/stamper"
	"github.com/spf13/cobra"
)

const (
	optionNamePeriodicCheck string = "periodic-check"
	optionNameNamespace     string = "namespace"
	optionNameLabelSelector string = "label-selector"
	optionNameNodeGroups    string = "node-groups" // We are using optionNameNodeGroups and optionNameLabelSelector in /cmd/beekeeper/cmd/cmd.go, i think we should move them to some common place ?
)

func (c *command) initStamperCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "stamper",
		Short: "manage postage batches for nodes",
		Long: `Manages postage batches for Bee nodes in your cluster or namespace.

The stamper command provides comprehensive postage batch management with subcommands:
• create: Generate new postage batches with specified depth and duration
• topup: Extend the TTL of existing postage batches before they expire
• dilute: Increase batch depth when usage approaches capacity limits
• set: Configure all postage batch parameters in one operation

Postage batches are essential for:
• Data uploads to the Swarm network
• Managing storage costs and capacity
• Ensuring data persistence and availability

Use --cluster-name or --namespace to target specific nodes.
Use --label-selector to filter nodes within a namespace.
Use --batch-ids or --postage-labels to target specific batches.

Each subcommand supports periodic execution for automated batch management.`,
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
		optionNameTTLThreshold  = "ttl-threshold"
		optionNameTopUpTo       = "topup-to"
		optionNameBatchIDs      = "batch-ids"
		optionNamePostageLabels = "postage-labels"
	)

	cmd := &cobra.Command{
		Use:   "topup",
		Short: "Top up the TTL of postage batches",
		Long: `Extends the Time-To-Live (TTL) of postage batches before they expire.

The topup command monitors postage batches and automatically extends their TTL when it
drops below the --ttl-threshold. This ensures continuous data availability and prevents
data loss due to expired postage.

Use --topup-to to specify how long to extend the TTL (default: 30 days).
Use --ttl-threshold to set when topup should occur (default: 5 days remaining).
Use --batch-ids or --postage-labels to target specific batches.
Use --periodic-check for continuous monitoring and automatic topup.`,
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
						stamper.WithPostageLabels(c.globalConfig.GetStringSlice(optionNamePostageLabels)),
					)
				})
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Duration(optionNameTTLThreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionNameTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().StringSlice(optionNameBatchIDs, nil, "Comma separated list of postage batch IDs to top up. If not provided, all batches are topped up. Overrides postage labels.")
	cmd.Flags().StringSlice(optionNamePostageLabels, nil, "Comma separated list of postage labels to top up. If not provided, all batches are topped up.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

	return cmd
}

func (c *command) initStamperDilute() *cobra.Command {
	const (
		optionNameUsageThreshold = "usage-threshold"
		optionNameDiutionDepth   = "dilution-depth"
		optionNameBatchIDs       = "batch-ids"
		optionNamePostageLabels  = "postage-labels"
	)

	cmd := &cobra.Command{
		Use:   "dilute",
		Short: "Dilute postage batches",
		Long: `Increases the depth of postage batches when usage approaches capacity limits.

The dilute command monitors postage batch usage and automatically increases their depth
when utilization exceeds the --usage-threshold. This prevents batch exhaustion and
ensures continuous data upload capability.

Use --usage-threshold to set when dilution should occur (default: 90% usage).
Use --dilution-depth to specify how many levels to increase depth by (default: 1).
Use --batch-ids or --postage-labels to target specific batches.
Use --periodic-check for continuous monitoring and automatic dilution.

Dilution increases the number of chunks that can be signed with the batch.`,
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
						stamper.WithPostageLabels(c.globalConfig.GetStringSlice(optionNamePostageLabels)),
					)
				})
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Float64(optionNameUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint8(optionNameDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")
	cmd.Flags().StringSlice(optionNameBatchIDs, nil, "Comma separated list of postage batch IDs to dilute. If not provided, all batches are diluted. Overrides postage labels.")
	cmd.Flags().StringSlice(optionNamePostageLabels, nil, "Comma separated list of postage labels to top up. If not provided, all batches are topped up.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

	return cmd
}

func (c *command) initStamperCreate() *cobra.Command {
	const (
		optionNameDuration     = "duration"
		optionNameDepth        = "depth"
		optionNamePostageLabel = "postage-label"
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a postage batch for selected nodes",
		Long: `Creates new postage batches for Bee nodes in your cluster or namespace.

The create command generates postage batches with specified parameters:
• --depth: Sets the batch depth (logarithmic scale for chunk capacity)
• --duration: Sets the Time-To-Live (TTL) for the batch
• --postage-label: Assigns a label for easy identification

Batch depth determines how many chunks can be signed:
• Depth 16: 65,536 chunks (default bucket depth)
• Depth 17: 131,072 chunks (recommended minimum)
• Higher depths provide more capacity but cost more

Use --cluster-name to target all nodes in a Beekeeper cluster.
Use --namespace with --label-selector to target specific nodes.
Postage batches are essential for data uploads to the Swarm network.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				stamperClient, err := c.createStamperClient(ctx)
				if err != nil {
					return fmt.Errorf("failed to create stamper client: %w", err)
				}

				return stamperClient.Create(ctx,
					c.globalConfig.GetDuration(optionNameDuration),
					c.globalConfig.GetUint16(optionNameDepth),
					c.globalConfig.GetString(optionNamePostageLabel),
				)
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().Duration(optionNameDuration, 24*time.Hour, "Duration of the postage batch")
	cmd.Flags().Uint16(optionNameDepth, 17, "Batch depth which specifies how many chunks can be signed with the batch. It is a logarithm. Must be higher than default bucket depth (16)")
	cmd.Flags().String(optionNamePostageLabel, "beekeeper", "Postage label for the batch")

	return cmd
}

func (c *command) initStamperSet() *cobra.Command {
	const (
		optionNameTTLThreshold   = "ttl-threshold"
		optionNameTopUpTo        = "topup-to"
		optionNameUsageThreshold = "usage-threshold"
		optionNameDiutionDepth   = "dilution-depth"
		optionNameBatchIDs       = "batch-ids"
		optionNamePostageLabels  = "postage-labels"
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set stamper configuration",
		Long: `Configures comprehensive postage batch management settings for selected nodes.

The set command combines topup, dilution, and monitoring in one operation:
• Automatically tops up TTL when it drops below --ttl-threshold
• Automatically dilutes batches when usage exceeds --usage-threshold
• Runs continuously with --periodic-check for automated management

Configuration options:
• --ttl-threshold: When to trigger TTL topup (default: 5 days remaining)
• --topup-to: How long to extend TTL (default: 30 days)
• --usage-threshold: When to trigger dilution (default: 90% usage)
• --dilution-depth: How many levels to increase depth by (default: 1)

Use --cluster-name or --namespace to target nodes.
Use --batch-ids or --postage-labels to target specific batches.
This command provides a complete solution for automated postage batch management.`,
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
						stamper.WithPostageLabels(c.globalConfig.GetStringSlice(optionNamePostageLabels)),
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
	cmd.Flags().StringSlice(optionNameBatchIDs, nil, "Comma separated list of postage batch IDs to set. If not provided, all batches are set. Overrides postage labels.")
	cmd.Flags().StringSlice(optionNamePostageLabels, nil, "Comma separated list of postage labels to set. If not provided, all batches are set.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0, "Periodic check interval. Default is 0, which means no periodic check.")

	return cmd
}

func (c *command) createStamperClient(ctx context.Context) (*stamper.Client, error) {
	nodeClient, err := c.createNodeClient(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("creating node client: %w", err)
	}

	return stamper.New(&stamper.ClientConfig{
		Log:        c.log,
		SwapClient: c.swapClient,
		NodeClient: nodeClient,
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

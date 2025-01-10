package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
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

			periodicCheck := c.globalConfig.GetDuration(optionNamePeriodicCheck)

			if periodicCheck == 0 {
				return c.stamper.Topup(ctx, c.globalConfig.GetDuration(optionTTLThreshold), c.globalConfig.GetDuration(optionTopUpTo))
			}

			periodicExecutor := scheduler.NewPeriodicExecutor(periodicCheck, c.log)
			periodicExecutor.Start(ctx, func(ctx context.Context) error {
				return c.stamper.Topup(ctx, c.globalConfig.GetDuration(optionTTLThreshold), c.globalConfig.GetDuration(optionTopUpTo))
			})
			defer func() {
				if err := periodicExecutor.Close(); err != nil {
					c.log.Errorf("failed to close topup periodic executor: %v", err)
				}
			}()

			<-ctx.Done()

			c.log.Infof("topup stopped: %v", ctx.Err())

			return nil
		},
		PreRunE: c.preRunE,
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

			periodicCheck := c.globalConfig.GetDuration(optionNamePeriodicCheck)

			if periodicCheck == 0 {
				return c.stamper.Dilute(ctx, c.globalConfig.GetFloat64(optionUsageThreshold), c.globalConfig.GetUint16(optionDiutionDepth))
			}

			periodicExecutor := scheduler.NewPeriodicExecutor(periodicCheck, c.log)
			periodicExecutor.Start(ctx, func(ctx context.Context) error {
				return c.stamper.Dilute(ctx, c.globalConfig.GetFloat64(optionUsageThreshold), c.globalConfig.GetUint16(optionDiutionDepth))
			})
			defer func() {
				if err := periodicExecutor.Close(); err != nil {
					c.log.Errorf("failed to close dilution periodic executor: %v", err)
				}
			}()

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

	cmd.Flags().Uint64(optionNameAmount, 100000000, "Amount of tokens to be staked in the postage batch.")
	cmd.Flags().Uint16(optionNameDepth, 0, "Depth of the postage batch.")

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

			periodicCheck := c.globalConfig.GetDuration(optionNamePeriodicCheck)

			setFunc := func(ctx context.Context) error {
				return c.stamper.Set(ctx,
					c.globalConfig.GetDuration(optionTTLThreshold),
					c.globalConfig.GetDuration(optionTopUpTo),
					c.globalConfig.GetFloat64(optionUsageThreshold),
					c.globalConfig.GetUint16(optionDiutionDepth),
				)
			}

			if periodicCheck == 0 {
				return setFunc(ctx)
			}

			periodicExecutor := scheduler.NewPeriodicExecutor(periodicCheck, c.log)
			periodicExecutor.Start(ctx, func(ctx context.Context) error {
				return setFunc(ctx)
			})
			defer func() {
				if err := periodicExecutor.Close(); err != nil {
					c.log.Errorf("failed to close topup and dilute periodic executor: %v", err)
				}
			}()

			<-ctx.Done()

			c.log.Infof("topup and dilute stopped: %v", ctx.Err())

			return nil
		},
	}

	cmd.Flags().Duration(optionTTLThreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().Float64(optionUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint16(optionDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")

	c.root.AddCommand(cmd)

	return cmd
}

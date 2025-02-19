package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/ethersphere/beekeeper/pkg/tracing"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initCheckCmd() error {
	const (
		optionNameCreateCluster        = "create-cluster"
		optionNameChecks               = "checks"
		optionNameMetricsEnabled       = "metrics-enabled"
		optionNameSeed                 = "seed"
		optionNameTimeout              = "timeout"
		optionNameMetricsPusherAddress = "metrics-pusher-address"
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "runs integration tests on a Bee cluster",
		Long:  `runs integration tests on a Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			checks := c.globalConfig.GetStringSlice(optionNameChecks)
			if len(checks) == 0 {
				return fmt.Errorf("no checks provided")
			}

			clusterName := c.globalConfig.GetString(optionNameClusterName)
			if clusterName == "" {
				return errMissingClusterName
			}

			// set cluster config
			cfgCluster, ok := c.config.Clusters[clusterName]
			if !ok {
				return fmt.Errorf("cluster %s not defined", clusterName)
			}

			cluster, err := c.setupCluster(ctx, clusterName, c.globalConfig.GetBool(optionNameCreateCluster))
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			var (
				metricsPusher  *push.Pusher
				metricsEnabled = c.globalConfig.GetBool(optionNameMetricsEnabled)
				cleanup        func()
			)

			if metricsEnabled {
				metricsPusher, cleanup = newMetricsPusher(c.globalConfig.GetString(optionNameMetricsPusherAddress), cfgCluster.GetNamespace(), c.log)
				// cleanup executes when the calling context terminates
				defer cleanup()
			}

			// logger metrics
			if l, ok := c.log.(metrics.Reporter); ok && metricsEnabled {
				metrics.RegisterCollectors(metricsPusher, l.Report()...)
			}

			// tracing
			tracingEndpoint := c.globalConfig.GetString(optionNameTracingEndpoint)
			if c.globalConfig.IsSet(optionNameTracingHost) && c.globalConfig.IsSet(optionNameTracingPort) {
				tracingEndpoint = strings.Join([]string{c.globalConfig.GetString(optionNameTracingHost), c.globalConfig.GetString(optionNameTracingPort)}, ":")
			}
			tracer, tracerCloser, err := tracing.NewTracer(&tracing.Options{
				Enabled:     c.globalConfig.GetBool(optionNameTracingEnabled),
				Endpoint:    tracingEndpoint,
				ServiceName: c.globalConfig.GetString(optionNameTracingServiceName),
			})
			if err != nil {
				return fmt.Errorf("tracer: %w", err)
			}
			defer tracerCloser.Close()

			// set global config
			checkGlobalConfig := config.CheckGlobalConfig{
				Seed:    c.globalConfig.GetInt64(optionNameSeed),
				GethURL: c.globalConfig.GetString(optionNameGethURL),
			}

			checkRunner := check.NewCheckRunner(checkGlobalConfig, c.config.Checks, cluster, metricsPusher, tracer, c.log)

			return checkRunner.Run(ctx, checks)
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "cluster name. Required")
	cmd.Flags().String(optionNameMetricsPusherAddress, "pushgateway.staging.internal", "prometheus metrics pusher address")
	cmd.Flags().Bool(optionNameCreateCluster, false, "creates cluster before executing checks")
	cmd.Flags().StringSlice(optionNameChecks, []string{"pingpong"}, "list of checks to execute")
	cmd.Flags().Bool(optionNameMetricsEnabled, true, "enable metrics")
	cmd.Flags().Int64(optionNameSeed, -1, "seed, -1 for random")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

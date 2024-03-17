package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/ethersphere/beekeeper/pkg/tracing"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initCheckCmd() (err error) {
	const (
		optionNameClusterName          = "cluster-name"
		optionNameCreateCluster        = "create-cluster"
		optionNameChecks               = "checks"
		optionNameMetricsEnabled       = "metrics-enabled"
		optionNameSeed                 = "seed"
		optionNameTimeout              = "timeout"
		optionNameMetricsPusherAddress = "metrics-pusher-address"
		// TODO: optionNameStages         = "stages"
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "runs integration tests on a Bee cluster",
		Long:  `runs integration tests on a Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			// set cluster config
			cfgCluster, ok := c.config.Clusters[c.globalConfig.GetString(optionNameClusterName)]
			if !ok {
				return fmt.Errorf("cluster %s not defined", c.globalConfig.GetString(optionNameClusterName))
			}

			// setup cluster
			cluster, err := c.setupCluster(ctx,
				c.globalConfig.GetString(optionNameClusterName),
				c.config,
				c.globalConfig.GetBool(optionNameCreateCluster),
				c.globalConfig.GetBool(optionNameEnableK8S),
			)
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
				Seed: c.globalConfig.GetInt64(optionNameSeed),
			}

			// run checks
			for _, checkName := range c.globalConfig.GetStringSlice(optionNameChecks) {
				checkName = strings.TrimSpace(checkName)
				// get configuration
				checkConfig, ok := c.config.Checks[checkName]
				if !ok {
					return fmt.Errorf("check '%s' doesn't exist", checkName)
				}

				// choose check type
				check, ok := config.Checks[checkConfig.Type]
				if !ok {
					return fmt.Errorf("check %s not implemented", checkConfig.Type)
				}

				// create check options
				o, err := check.NewOptions(checkGlobalConfig, checkConfig)
				if err != nil {
					return fmt.Errorf("creating check %s options: %w", checkName, err)
				}

				// create check
				chk := check.NewAction(c.log)
				if r, ok := chk.(metrics.Reporter); ok && metricsEnabled {
					metrics.RegisterCollectors(metricsPusher, r.Report()...)
				}
				chk = beekeeper.NewActionMiddleware(tracer, chk, checkName)

				if checkConfig.Timeout != nil {
					ctx, cancel = context.WithTimeout(ctx, *checkConfig.Timeout)
					defer cancel()
				}

				c.log.Infof("running check: %s", checkName)

				ch := make(chan error, 1)
				go func() {
					ch <- chk.Run(ctx, cluster, o)
					close(ch)
				}()

				select {
				case <-ctx.Done():
					deadline, ok := ctx.Deadline()
					if ok {
						return fmt.Errorf("running check %s: %w: deadline %v", checkName, ctx.Err(), deadline)
					}
					return fmt.Errorf("running check %s: %w", checkName, ctx.Err())
				case err = <-ch:
					if err != nil {
						return fmt.Errorf("running check %s: %w", checkName, err)
					}
					c.log.Infof("%s check completed successfully", checkName)
				}
			}
			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")
	cmd.Flags().String(optionNameMetricsPusherAddress, "pushgateway.staging.internal", "prometheus metrics pusher address")
	cmd.Flags().Bool(optionNameCreateCluster, false, "creates cluster before executing checks")
	cmd.Flags().StringSlice(optionNameChecks, []string{"pingpong"}, "list of checks to execute")
	cmd.Flags().Bool(optionNameMetricsEnabled, true, "enable metrics")
	cmd.Flags().Int64(optionNameSeed, -1, "seed, -1 for random")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

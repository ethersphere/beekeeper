package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initSimulateCmd() (err error) {
	const (
		optionNameClusterName          = "cluster-name"
		optionNameCreateCluster        = "create-cluster"
		optionNameSimulations          = "simulations"
		optionNameMetricsEnabled       = "metrics-enabled"
		optionNameSeed                 = "seed"
		optionNameTimeout              = "timeout"
		optionNameMetricsPusherAddress = "metrics-pusher-address"
		// TODO: optionNameStages         = "stages"
	)

	cmd := &cobra.Command{
		Use:   "simulate",
		Short: "runs simulations on a Bee cluster",
		Long:  `Runs simulations on a Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			// set cluster config
			cfgCluster, ok := c.config.Clusters[c.globalConfig.GetString(optionNameClusterName)]
			if !ok {
				return fmt.Errorf("cluster %s not defined", c.globalConfig.GetString(optionNameClusterName))
			}

			// setup cluster
			cluster, err := c.setupCluster(ctx, c.globalConfig.GetString(optionNameClusterName), c.config, c.globalConfig.GetBool(optionNameCreateCluster))
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			var (
				metricsPusher  *push.Pusher
				metricsEnabled = c.globalConfig.GetBool(optionNameMetricsEnabled)
				cleanup        func()
			)

			if metricsEnabled {
				metricsPusher, cleanup = newMetricsPusher(c.globalConfig.GetString(optionNameMetricsPusherAddress), cfgCluster.GetNamespace())

				// cleanup executes when the calling context terminates
				defer cleanup()
			}

			// set global config
			simulationGlobalConfig := config.SimulationGlobalConfig{
				Seed: c.globalConfig.GetInt64(optionNameSeed),
			}

			// run simulations
			for _, simulationName := range c.globalConfig.GetStringSlice(optionNameSimulations) {
				// get configuration
				simulationConfig, ok := c.config.Simulations[simulationName]
				if !ok {
					return fmt.Errorf("simulation %s doesn't exist", simulationName)
				}

				// choose simulation type
				simulation, ok := config.Simulations[simulationConfig.Type]
				if !ok {
					return fmt.Errorf("simulation %s not implemented", simulationConfig.Type)
				}

				// create simulation options
				o, err := simulation.NewOptions(simulationGlobalConfig, simulationConfig)
				if err != nil {
					return fmt.Errorf("creating simulation %s options: %w", simulationName, err)
				}

				// create simulation
				sim := simulation.NewAction()
				if s, ok := sim.(metrics.Reporter); ok && metricsEnabled {
					metrics.RegisterCollectors(metricsPusher, s.Report()...)
				}

				// run simulation
				if err := sim.Run(ctx, cluster, o); err != nil {
					return fmt.Errorf("running simulation %s: %w", simulationName, err)
				}
			}

			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")
	cmd.Flags().String(optionNameMetricsPusherAddress, "pushgateway.dai.internal", "prometheus metrics pusher address")
	cmd.Flags().Bool(optionNameCreateCluster, false, "creates cluster before executing simulations")
	cmd.Flags().StringSlice(optionNameSimulations, []string{"upload"}, "list of simulations to execute")
	cmd.Flags().Bool(optionNameMetricsEnabled, true, "enable metrics")
	cmd.Flags().Int64(optionNameSeed, -1, "seed, -1 for random")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

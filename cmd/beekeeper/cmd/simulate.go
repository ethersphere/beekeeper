package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initSimulateCmd() (err error) {
	const (
		optionNameClusterName    = "cluster-name"
		optionNameCreateCluster  = "create-cluster"
		optionNameSimulations    = "simulations"
		optionNameMetricsEnabled = "metrics-enabled"
		optionNameSeed           = "seed"
		// optionNameStages         = "stages"
		// optionNameTimeout        = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "simulate",
		Short: "Run simulations on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config/config.yaml")
			if err != nil {
				return err
			}

			cfgCluster, ok := cfg.Clusters[c.config.GetString(optionNameClusterName)]
			if !ok {
				return fmt.Errorf("cluster %s not defined", c.config.GetString(optionNameClusterName))
			}

			simulationGlobalConfig := config.SimulationGlobalConfig{
				MetricsEnabled: c.config.GetBool(optionNameMetricsEnabled),
				MetricsPusher:  push.New("beekeeper", *cfgCluster.Namespace),
				Seed:           c.config.GetInt64(optionNameSeed),
			}

			cluster, err := c.setupCluster(cmd.Context(), c.config.GetString(optionNameClusterName), cfg, c.config.GetBool(optionNameCreateCluster))
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			for _, simulationName := range c.config.GetStringSlice(optionNameSimulations) {
				simulationConfig, ok := cfg.Simulations[simulationName]
				if !ok {
					return fmt.Errorf("simulation %s doesn't exist", simulationName)
				}

				simulation, ok := config.Simulations[simulationConfig.Type]
				if !ok {
					return fmt.Errorf("simulation %s not implemented", simulationConfig.Type)
				}

				o, err := simulation.NewOptions(simulationGlobalConfig, simulationConfig)
				if err != nil {
					return fmt.Errorf("creating simulation %s options: %w", simulationName, err)
				}

				if err := simulation.NewAction().Run(cmd.Context(), cluster, o); err != nil {
					return fmt.Errorf("running simulation %s: %w", simulationName, err)
				}
			}

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")
	cmd.Flags().Bool(optionNameCreateCluster, false, "start cluster")
	cmd.Flags().StringSlice(optionNameSimulations, []string{"upload"}, "simulations")
	cmd.Flags().Bool(optionNameMetricsEnabled, false, "enable metrics")
	cmd.Flags().Int64(optionNameSeed, 0, "seed")

	c.root.AddCommand(cmd)

	return nil
}

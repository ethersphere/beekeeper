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

	var (
		clusterName    string
		creaateCluster bool
		simulations    []string
		metricsEnabled bool
		seed           int64
		// stages string
		// timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "simulate",
		Short: "Run simulation on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}

			cfgCluster, ok := cfg.Clusters[clusterName]
			if !ok {
				return fmt.Errorf("cluster %s not defined", clusterName)
			}

			globalSimulationConfig := config.GlobalSimulationConfig{
				MetricsEnabled: metricsEnabled,
				MetricsPusher:  push.New("beekeeper", cfgCluster.Namespace),
				Seed:           seed,
			}

			cluster, err := setupCluster(cmd.Context(), clusterName, cfg, creaateCluster)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			for _, simulation := range simulations {
				cfgSimulation, ok := cfg.Simulations[simulation]
				if !ok {
					return fmt.Errorf("simulation %s doesn't exist", simulation)
				}

				simulation, ok := config.Simulations[cfgSimulation.Name]
				if !ok {
					return fmt.Errorf("simulation %s not implemented", cfgSimulation.Name)
				}

				o, err := simulation.NewOptions(cfgSimulation, globalSimulationConfig)
				if err != nil {
					return fmt.Errorf("creating simulation %s options: %w", cfgSimulation.Name, err)
				}

				if err := simulation.NewAction().Run(cmd.Context(), cluster, o); err != nil {
					return fmt.Errorf("running simulation %s: %w", cfgSimulation.Name, err)
				}
			}

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "default", "cluster name")
	cmd.Flags().BoolVar(&creaateCluster, optionNameCreateCluster, false, "start cluster")
	cmd.Flags().StringArrayVar(&simulations, optionNameSimulations, []string{"upload"}, "simulations")
	cmd.Flags().BoolVar(&metricsEnabled, optionNameMetricsEnabled, false, "enable metrics")
	cmd.Flags().Int64Var(&seed, optionNameSeed, 0, "seed")

	c.root.AddCommand(cmd)

	return nil
}

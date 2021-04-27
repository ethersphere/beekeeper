package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initCheckCmd() (err error) {
	const (
		optionNameClusterName    = "cluster-name"
		optionNameCreateCluster  = "create-cluster"
		optionNameChecks         = "checks"
		optionNameMetricsEnabled = "metrics-enabled"
		optionNameSeed           = "seed"
		// optionNameStages         = "stages"
		// optionNameTimeout        = "timeout"
	)

	var (
		clusterName    string
		creaateCluster bool
		checks         []string
		metricsEnabled bool
		seed           int64
		// stages string
		// timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run tests on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}

			cfgCluster, ok := cfg.Clusters[clusterName]
			if !ok {
				return fmt.Errorf("cluster %s not defined", clusterName)
			}

			globalCheckConfig := config.GlobalCheckConfig{
				MetricsEnabled: metricsEnabled,
				MetricsPusher:  push.New("beekeeper", cfgCluster.Namespace),
				Seed:           seed,
			}

			cluster, err := setupCluster(cmd.Context(), clusterName, cfg, creaateCluster)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			for _, checkName := range checks {
				cfgCheck, ok := cfg.Checks[checkName]
				if !ok {
					return fmt.Errorf("check %s doesn't exist", checkName)
				}

				check, ok := config.Checks[cfgCheck.Name]
				if !ok {
					return fmt.Errorf("check %s not implemented", cfgCheck.Name)
				}

				o, err := check.NewOptions(cfgCheck, globalCheckConfig)
				if err != nil {
					return fmt.Errorf("creating check %s options: %w", cfgCheck.Name, err)
				}

				if err := check.NewAction().Run(cmd.Context(), cluster, o); err != nil {
					return fmt.Errorf("running check %s: %w", cfgCheck.Name, err)
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
	cmd.Flags().StringArrayVar(&checks, optionNameChecks, []string{"pingpong"}, "checks")
	cmd.Flags().BoolVar(&metricsEnabled, optionNameMetricsEnabled, false, "enable metrics")
	cmd.Flags().Int64Var(&seed, optionNameSeed, 0, "seed")

	c.root.AddCommand(cmd)

	return nil
}

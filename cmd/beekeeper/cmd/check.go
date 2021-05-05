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

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run tests on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config/config.yaml")
			if err != nil {
				return err
			}

			cfgCluster, ok := cfg.Clusters[c.config.GetString(optionNameClusterName)]
			if !ok {
				return fmt.Errorf("cluster %s not defined", c.config.GetString(optionNameClusterName))
			}

			checkGlobalConfig := config.CheckGlobalConfig{
				MetricsEnabled: c.config.GetBool(optionNameMetricsEnabled),
				MetricsPusher:  push.New("beekeeper", cfgCluster.GetNamespace()),
				Seed:           c.config.GetInt64(optionNameSeed),
			}

			cluster, err := c.setupCluster(cmd.Context(), c.config.GetString(optionNameClusterName), cfg, c.config.GetBool(optionNameCreateCluster))
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			for _, checkName := range c.config.GetStringSlice(optionNameChecks) {
				checkConfig, ok := cfg.Checks[checkName]
				if !ok {
					return fmt.Errorf("check %s doesn't exist", checkName)
				}

				check, ok := config.Checks[checkConfig.Type]
				if !ok {
					return fmt.Errorf("check %s not implemented", checkConfig.Type)
				}

				o, err := check.NewOptions(checkGlobalConfig, checkConfig)
				if err != nil {
					return fmt.Errorf("creating check %s options: %w", checkName, err)
				}

				if err := check.NewAction().Run(cmd.Context(), cluster, o); err != nil {
					return fmt.Errorf("running check %s: %w", checkName, err)
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
	cmd.Flags().StringArray(optionNameChecks, []string{"pingpong"}, "checks")
	cmd.Flags().Bool(optionNameMetricsEnabled, false, "enable metrics")
	cmd.Flags().Int64(optionNameSeed, 0, "seed")

	c.root.AddCommand(cmd)

	return nil
}

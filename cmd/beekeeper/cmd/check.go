package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initCheckCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run tests on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}

			cluster, ok := cfg.Clusters[cfg.Execute.Cluster]
			if !ok {
				return fmt.Errorf("cluster %s not defined", cfg.Execute.Cluster)
			}

			playbook, ok := cfg.Playbooks[cfg.Execute.Playbook]
			if !ok {
				return fmt.Errorf("playbook %s not defined", cfg.Execute.Playbook)
			}

			globalCheckConfig := config.GlobalCheckConfig{
				MetricsEnabled: playbook.ChecksGlobalConfig.MetricsEnabled,
				MetricsPusher:  push.New("beekeeper", cluster.Namespace),
				Seed:           playbook.ChecksGlobalConfig.Seed,
			}

			for _, checkName := range playbook.Checks {
				checkProfile, ok := cfg.Checks[checkName]
				if !ok {
					return fmt.Errorf("check %s doesn't exist", checkName)
				}

				check, ok := config.Checks[checkProfile.Name]
				if !ok {
					return fmt.Errorf("check %s not implemented", checkProfile.Name)
				}

				cluster, err := setupCluster(cmd.Context(), cfg, startCluster)
				if err != nil {
					return fmt.Errorf("cluster setup: %w", err)
				}

				o, err := check.NewOptions(checkProfile, globalCheckConfig)
				if err != nil {
					return fmt.Errorf("creating check %s options: %w", checkProfile.Name, err)
				}

				if err := check.NewCheck().Run(cmd.Context(), cluster, o); err != nil {
					return fmt.Errorf("running check %s: %w", checkProfile.Name, err)
				}
			}

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	c.root.AddCommand(cmd)
	return nil
}

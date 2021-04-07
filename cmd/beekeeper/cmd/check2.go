package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initCheck2Cmd() (err error) {
	const (
		optionNameStartCluster = "start-cluster"
	)

	var (
		startCluster bool
	)

	cmd := &cobra.Command{
		Use:   "check2",
		Short: "Run tests on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")

			for _, checkName := range cfg.Run["default"].Checks {
				checkProfile, ok := cfg.Checks[checkName]
				if !ok {
					return fmt.Errorf("check %s doesn't exist", checkName)
				}

				check, ok := Checks[checkProfile.Name]
				if !ok {
					return fmt.Errorf("check %s not implemented", checkProfile.Name)
				}

				cluster, err := setupCluster(cmd.Context(), cfg, startCluster)
				if err != nil {
					return fmt.Errorf("cluster setup: %w", err)
				}

				o, err := check.NewOptions(cfg, checkProfile)
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

	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")

	c.root.AddCommand(cmd)
	return nil
}

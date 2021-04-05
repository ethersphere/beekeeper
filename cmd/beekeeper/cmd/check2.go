package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/check/ping"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

type pingOptions struct {
	MetricsEnabled *bool  `yaml:"metrics-enabled"`
	Seed           *int64 `yaml:"seed"`
}

type Check struct {
	Constructor func() check.Check
	NewOptions  func(cfg *config.Config, checkProfile config.CheckProfile) (interface{}, error)
}

var Checks = map[string]Check{
	"ping": {
		Constructor: ping.NewPing,
		NewOptions: func(cfg *config.Config, checkProfile config.CheckProfile) (interface{}, error) {
			o := new(pingOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts ping.Options
			if o.Seed != nil {
				opts.Seed = *o.Seed
			}
			if o.MetricsEnabled != nil && *o.MetricsEnabled {
				// TODO: make pusher and set it to opts.MetricsPusher
			}
			return opts, nil
		},
	},
}

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

			for _, checkName := range cfg.Run.Checks {
				checkProfile, ok := cfg.CheckProfiles[checkName]
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

				if err := check.Constructor().Run(cmd.Context(), cluster, o); err != nil {
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

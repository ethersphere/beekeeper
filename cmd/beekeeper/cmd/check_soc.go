package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/soc"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckSOC() *cobra.Command {
	const (
		optionNameSeed = "seed"
	)

	cmd := &cobra.Command{
		Use:   "soc",
		Short: "Checks SOC ability of the cluster",
		Long:  `Checks SOC ability of the cluster. First a SOC is uploaded and then retrieved using the returned reference`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			return soc.Check(cluster, soc.Options{
				NodeGroup: "bee",
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for choosing random nodes; if not set, will be random")

	return cmd
}

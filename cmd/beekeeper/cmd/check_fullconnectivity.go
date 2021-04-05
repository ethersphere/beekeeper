package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
	"github.com/ethersphere/beekeeper/pkg/config"

	"github.com/spf13/cobra"
)

func (c *command) initCheckFullConnectivity() *cobra.Command {
	return &cobra.Command{
		Use:   "fullconnectivity",
		Short: "Checks full connectivity in the cluster",
		Long:  `Checks if every node has connectivity to all other nodes in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			return fullconnectivity.Check(cmd.Context(), cluster)
		},
		PreRunE: c.checkPreRunE,
	}
}

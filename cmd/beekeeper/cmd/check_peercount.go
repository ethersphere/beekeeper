package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/peercount"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initCheckPeerCount() *cobra.Command {
	return &cobra.Command{
		Use:   "peercount",
		Short: "Counts peers for all nodes in the cluster",
		Long:  `Counts peers for all nodes in the cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			return peercount.Check(cluster)
		},
		PreRunE: c.checkPreRunE,
	}
}

package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initPrintOverlay() *cobra.Command {
	return &cobra.Command{
		Use:   "overlays",
		Short: "Print overlay addresses",
		Long:  `Print overlay address for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			overlays, err := cluster.Overlays(cmd.Context())
			if err != nil {
				return err
			}

			for ng, no := range overlays {
				fmt.Printf("Printing %s node group's overlays\n", ng)
				for n, o := range no {
					fmt.Printf("Node %s. %s\n", n, o.String())
				}
			}

			return
		},
		PreRunE: c.printPreRunE,
	}
}

package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initPrintDepths() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "depths",
		Short: "Print kademlia depths",
		Long:  `Print list of Kademlia depths for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			topologies, err := cluster.Topologies(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Println(topologies)

			for ng, nt := range topologies {
				fmt.Printf("Printing %s node group's topologies\n", ng)
				for n, t := range nt {
					fmt.Printf("Node %s. overlay: %s depth: %d\n", n, t.Overlay, t.Depth)
				}
			}

			return
		},
		PreRunE: c.printPreRunE,
	}

	return cmd
}

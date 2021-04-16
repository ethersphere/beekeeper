package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initPrintTopologies() *cobra.Command {
	return &cobra.Command{
		Use:   "topologies",
		Short: "Print topologies",
		Long:  `Print list of Kademlia topology for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			topologies, err := cluster.Topologies(cmd.Context())
			if err != nil {
				return err
			}

			for ng, nt := range topologies {
				fmt.Printf("Printing %s node group's topologies\n", ng)
				for n, t := range nt {
					fmt.Printf("Node %s. overlay: %s\n", n, t.Overlay)
					fmt.Printf("Node %s. population: %d\n", n, t.Population)
					fmt.Printf("Node %s. connected: %d\n", n, t.Connected)
					fmt.Printf("Node %s. depth: %d\n", n, t.Depth)
					fmt.Printf("Node %s. nnLowWatermark: %d\n", n, t.NnLowWatermark)
					for k, v := range t.Bins {
						fmt.Printf("Node %s. %s %+v\n", n, k, v)
					}
				}
			}

			return
		},
		PreRunE: c.printPreRunE,
	}
}

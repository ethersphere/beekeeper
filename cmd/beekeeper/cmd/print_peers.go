package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initPrintPeers() *cobra.Command {
	const (
		optionNameClusterName = "cluster-name"
	)
	var (
		clusterName string
	)
	return &cobra.Command{
		Use:   "peers",
		Short: "Print peers",
		Long:  `Print list of peers for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}
			cluster, err := setupCluster(cmd.Context(), clusterName, cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			peers, err := cluster.Peers(cmd.Context())
			if err != nil {
				return err
			}

			for ng, np := range peers {
				fmt.Printf("Printing %s node group's peers\n", ng)
				for n, a := range np {
					for _, p := range a {
						fmt.Printf("Node %s. %s\n", n, p)
					}
				}
			}

			return
		},
		PreRunE: c.printPreRunE,
	}
}

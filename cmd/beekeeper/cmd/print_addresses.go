package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initPrintAddresses() *cobra.Command {
	const (
		optionNameClusterName = "cluster-name"
	)
	var (
		clusterName string
	)
	return &cobra.Command{
		Use:   "addresses",
		Short: "Print addresses",
		Long:  `Print address for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}
			cluster, err := setupCluster(cmd.Context(), clusterName, cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			addresses, err := cluster.Addresses(cmd.Context())
			if err != nil {
				return err
			}

			for ng, na := range addresses {
				fmt.Printf("Printing %s node group's addresses\n", ng)
				for n, a := range na {
					fmt.Printf("Node %s. ethereum: %s\n", n, a.Ethereum)
					fmt.Printf("Node %s. public key: %s\n", n, a.PublicKey)
					fmt.Printf("Node %s. overlay: %s\n", n, a.Overlay)
					for _, u := range a.Underlay {
						fmt.Printf("Node %s. underlay: %s\n", n, u)
					}
				}
			}

			return
		},
		PreRunE: c.printPreRunE,
	}
}

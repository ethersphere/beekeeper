package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initPrintTopologies() *cobra.Command {
	return &cobra.Command{
		Use:   "topologies",
		Short: "Print topologies",
		Long:  `Print list of Kademlia topology for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cluster, err := bee.NewCluster(bee.ClusterOptions{
				APIScheme:               c.config.GetString(optionNameAPIScheme),
				APIHostnamePattern:      c.config.GetString(optionNameAPIHostnamePattern),
				APIDomain:               c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:          insecureTLSAPI,
				DebugAPIScheme:          c.config.GetString(optionNameDebugAPIScheme),
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS:     insecureTLSDebugAPI,
				Namespace:               c.config.GetString(optionNameNamespace),
				Size:                    c.config.GetInt(optionNameNodeCount),
			})
			if err != nil {
				return err
			}

			ctx := context.Background()
			topologies, err := cluster.Topologies(ctx)
			if err != nil {
				return err
			}

			for i, t := range topologies {
				fmt.Printf("Node %d. overlay: %s\n", i, t.Overlay)
				fmt.Printf("Node %d. population: %d\n", i, t.Population)
				fmt.Printf("Node %d. connected: %d\n", i, t.Connected)
				fmt.Printf("Node %d. depth: %d\n", i, t.Depth)
				fmt.Printf("Node %d. nnLowWatermark: %d\n", i, t.NnLowWatermark)
				for k, v := range t.Bins {
					fmt.Printf("Node %d. %s %+v\n", i, k, v)
				}
			}

			return
		},
		PreRunE: c.printPreRunE,
	}
}

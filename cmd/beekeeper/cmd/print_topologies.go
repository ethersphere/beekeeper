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
			cluster := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				KubeconfigPath:      c.config.GetString(optionNameStartKubeconfig),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i)); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			ctx := context.Background()
			topologies, err := ng.Topologies(ctx)
			if err != nil {
				return err
			}

			for n, t := range topologies {
				fmt.Printf("Node %s. overlay: %s\n", n, t.Overlay)
				fmt.Printf("Node %s. population: %d\n", n, t.Population)
				fmt.Printf("Node %s. connected: %d\n", n, t.Connected)
				fmt.Printf("Node %s. depth: %d\n", n, t.Depth)
				fmt.Printf("Node %s. nnLowWatermark: %d\n", n, t.NnLowWatermark)
				for k, v := range t.Bins {
					fmt.Printf("Node %s. %s %+v\n", n, k, v)
				}
			}

			return
		},
		PreRunE: c.printPreRunE,
	}
}

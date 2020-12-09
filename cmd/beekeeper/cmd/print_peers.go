package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initPrintPeers() *cobra.Command {
	return &cobra.Command{
		Use:   "peers",
		Short: "Print peers",
		Long:  `Print list of peers for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cluster := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", *ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i)); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			peers, err := ng.Peers(cmd.Context())
			if err != nil {
				return err
			}

			for n, a := range peers {
				for _, p := range a {
					fmt.Printf("Node %s. %s\n", n, p)
				}
			}

			return
		},
		PreRunE: c.printPreRunE,
	}
}

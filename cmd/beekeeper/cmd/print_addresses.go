package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initPrintAddresses() *cobra.Command {
	return &cobra.Command{
		Use:   "addresses",
		Short: "Print addresses",
		Long:  `Print address for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cluster, err := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				InCluster:           c.config.GetBool(optionNameInCluster),
				KubeconfigPath:      c.config.GetString(optionNameKubeconfig),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})
			if err != nil {
				return fmt.Errorf("creating new Bee cluster: %v", err)
			}

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", *ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i)); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			addresses, err := ng.Addresses(cmd.Context())
			if err != nil {
				return err
			}

			for n, a := range addresses {
				fmt.Printf("Node %s. ethereum: %s\n", n, a.Ethereum)
				fmt.Printf("Node %s. public key: %s\n", n, a.PublicKey)
				fmt.Printf("Node %s. overlay: %s\n", n, a.Overlay)
				for _, u := range a.Underlay {
					fmt.Printf("Node %s. underlay: %s\n", n, u)
				}

			}

			return
		},
		PreRunE: c.printPreRunE,
	}
}

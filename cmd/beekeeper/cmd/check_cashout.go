package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/cashout"

	"github.com/spf13/cobra"
)

func (c *command) initCheckCashout() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cashout",
		Short: "Executes cashout check",
		Long:  `Executes cashout check.`,
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
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i), bee.NodeOptions{}); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			return cashout.Check(cluster, cashout.Options{
				NodeGroup: "nodes",
			})
		},
		PreRunE: c.checkPreRunE,
	}

	return cmd
}

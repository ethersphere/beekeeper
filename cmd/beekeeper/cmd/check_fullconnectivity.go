package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckFullConnectivity() *cobra.Command {
	return &cobra.Command{
		Use:   "fullconnectivity",
		Short: "Checks full connectivity in the cluster",
		Long:  `Checks full connectivity in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			nodes, err := bee.NewNNodes(
				c.config.GetString(optionNameAPIHostnamePattern),
				c.config.GetString(optionNameNamespace),
				c.config.GetString(optionNameAPIDomain),
				c.config.GetString(optionNameDebugAPIHostnamePattern),
				c.config.GetString(optionNameNamespace),
				c.config.GetString(optionNameDebugAPIDomain),
				disableNamespace,
				c.config.GetInt(optionNameNodeCount),
			)
			if err != nil {
				return err
			}

			return check.FullConnectivity(nodes)
		},
		PreRunE: c.checkPreRunE,
	}
}

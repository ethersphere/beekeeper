package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckFullConnectivity() *cobra.Command {
	return &cobra.Command{
		Use:   "fullconnectivity",
		Short: "Checks full connectivity in the cluster",
		Long:  `Checks full connectivity in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.FullConnectivity(check.FullConnectivityOptions{
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				DisableNamespace:        disableNamespace,
				Namespace:               c.config.GetString(optionNameNamespace),
				NodeCount:               c.config.GetInt(optionNameNodeCount),
			})
		},
		PreRunE: c.checkPreRunE,
	}
}

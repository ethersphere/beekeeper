package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"

	"github.com/spf13/cobra"
)

func (c *command) initCheckFullConnectivity() *cobra.Command {
	return &cobra.Command{
		Use:   "fullconnectivity",
		Short: "Checks full connectivity in the cluster",
		Long:  `Checks if every node has connectivity to all other nodes in the cluster.`,
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
				DisableNamespace:        disableNamespace,
				Namespace:               c.config.GetString(optionNameNamespace),
				Size:                    c.config.GetInt(optionNameNodeCount),
			})
			if err != nil {
				return err
			}

			return fullconnectivity.Check(cluster)
		},
		PreRunE: c.checkPreRunE,
	}
}

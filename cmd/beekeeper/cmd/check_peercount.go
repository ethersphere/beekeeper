package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/peercount"
	"github.com/spf13/cobra"
)

func (c *command) initCheckPeerCount() *cobra.Command {
	return &cobra.Command{
		Use:   "peercount",
		Short: "Counts peers for all nodes in the cluster",
		Long:  `Counts peers for all nodes in the cluster`,
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

			return peercount.Check(cluster)
		},
		PreRunE: c.checkPreRunE,
	}
}

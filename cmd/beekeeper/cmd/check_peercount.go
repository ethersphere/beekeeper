package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/spf13/cobra"
)

func (c *command) initCheckPeerCount() *cobra.Command {
	return &cobra.Command{
		Use:   "peercount",
		Short: "Checks node's peer count for all nodes in the cluster",
		Long: `Checks node's peer count for all nodes in the cluster.
It retrieves list of peers from node's Debug API (/peers endpoint).`,
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

			return check.PeerCount(nodes)
		},
		PreRunE: c.checkPreRunE,
	}
}

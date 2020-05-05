package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPeerCount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "peercount",
		Short: "Checks node's peer count for all nodes in the cluster",
		Long: `Checks node's peer count for all nodes in the cluster.
It retrieves list of peers from node's Debug API (/peers endpoint).`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PeerCount(check.PeerCountOptions{
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				Namespace:               c.config.GetString(optionNameNamespace),
				NodeCount:               c.config.GetInt(optionNameNodeCount),
			})
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	return cmd
}

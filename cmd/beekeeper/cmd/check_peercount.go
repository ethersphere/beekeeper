package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPeerCount() *cobra.Command {
	const (
		optionNameBootNodeCount   = "bootnode-count"
		optionNameNodeCount       = "node-count"
		optionNameNodeURLTemplate = "node-url-template"
	)

	cmd := &cobra.Command{
		Use:   "peercount",
		Short: "Check node's peer count",
		Long: `Check node's peer count for all nodes in the cluster.
Retrieves list of peers from node's Debug API (/peers endpoint).
Compares number of node's peers against expected peer count (node-count + bootnode-count - 1).`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PeerCount(check.PeerCountOptions{
				BootNodeCount:   c.config.GetInt(optionNameBootNodeCount),
				NodeCount:       c.config.GetInt(optionNameNodeCount),
				NodeURLTemplate: c.config.GetString(optionNameNodeURLTemplate),
			})
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().Int(optionNameBootNodeCount, 1, "bootnode count")
	cmd.Flags().Int(optionNameNodeCount, 1, "node count")
	cmd.Flags().String(optionNameNodeURLTemplate, "", "node URL template (e.g. http://bee-%d-debug.domain)")
	cmd.MarkFlagRequired(optionNameNodeURLTemplate)

	return cmd
}

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
		Short: "Check peer count",
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
	cmd.Flags().String(optionNameNodeURLTemplate, "", "node URL template")
	cmd.MarkFlagRequired(optionNameNodeURLTemplate)

	return cmd
}

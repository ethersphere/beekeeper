package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPeerCount() *cobra.Command {
	const (
		optionNameNodeCount       = "node-count"
		optionNameNodeURLTemplate = "node-url-template"
	)

	cmd := &cobra.Command{
		Use:   "peercount",
		Short: "Check peer count",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PeerCount(check.PeerCountOptions{
				NodeCount:       c.config.GetInt(optionNameNodeCount),
				NodeURLTemplate: c.config.GetString(optionNameNodeURLTemplate),
			})
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().Int(optionNameNodeCount, 1, "bee node count")
	cmd.Flags().String(optionNameNodeURLTemplate, "", "bee node URL template")

	return cmd
}

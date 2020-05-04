package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPeerCount() *cobra.Command {
	const (
		optionNameNodeCount   = "node-count"
		optionNameNamespace   = "namespace"
		optionNameURLTemplate = "url-template"
	)

	cmd := &cobra.Command{
		Use:   "peercount",
		Short: "Checks node's peer count for all nodes in the cluster",
		Long: `Checks node's peer count for all nodes in the cluster.
It retrieves list of peers from node's Debug API (/peers endpoint).`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PeerCount(check.PeerCountOptions{
				NodeCount:   c.config.GetInt(optionNameNodeCount),
				Namespace:   c.config.GetString(optionNameNamespace),
				URLTemplate: c.config.GetString(optionNameURLTemplate),
			})
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flag(optionNameURLTemplate).Value.String() == "" {
				if err := cmd.MarkFlagRequired(optionNameNamespace); err != nil {
					panic(err)
				}
			}
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().IntP(optionNameNodeCount, "c", 1, "node count")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace")
	cmd.Flags().StringP(optionNameURLTemplate, "u", "", "URL template")
	if err := cmd.Flags().MarkHidden(optionNameURLTemplate); err != nil {
		panic(err)
	}

	return cmd
}

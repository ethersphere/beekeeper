package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPeerCount() *cobra.Command {
	const (
		optionNameBootNodeCount = "bootnode-count"
		optionNameNodeCount     = "node-count"
		optionNameNamespace     = "namespace"
	)

	cmd := &cobra.Command{
		Use:   "peercount",
		Short: "Checks node's peer count for all nodes in the cluster",
		Long: `Checks node's peer count for all nodes in the cluster.
It retrieves list of peers from node's Debug API (/peers endpoint),
and compares number of node's peers against expected peer count.
Expected peer count equals: node-count + bootnode-count - 1.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PeerCount(check.PeerCountOptions{
				BootNodeCount: c.config.GetInt(optionNameBootNodeCount),
				NodeCount:     c.config.GetInt(optionNameNodeCount),
				Namespace:     c.config.GetString(optionNameNamespace),
			})
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().IntP(optionNameBootNodeCount, "b", 1, "bootnode count")
	cmd.Flags().IntP(optionNameNodeCount, "c", 1, "node count")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace")
	if err := cmd.MarkFlagRequired(optionNameNamespace); err != nil {
		panic(err)
	}

	return cmd
}

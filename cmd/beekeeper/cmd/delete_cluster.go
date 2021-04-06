package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initDeleteCluster() *cobra.Command {
	const (
		optionNameClusterName         = "cluster-name"
		optionNameBootnodeCount       = "bootnode-count"
		optionNameNodeCount           = "node-count"
		optionNameAdditionalNodeCount = "additional-node-count"
	)

	var (
		clusterName         string
		bootnodeCount       int
		nodeCount           int
		additionalNodeCount int
	)

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Delete Bee cluster",
		Long:  `Delete Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			return deleteCluster(cmd.Context(), cfg)
		},
		PreRunE: c.deletePreRunE,
	}

	cmd.PersistentFlags().StringVar(&clusterName, optionNameClusterName, "bee", "cluster name")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 1, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")

	return cmd
}

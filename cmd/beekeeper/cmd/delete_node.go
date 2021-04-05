package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *command) initDeleteNode() *cobra.Command {
	const (
		optionNameClusterName   = "cluster-name"
		optionNameNodeGroupName = "node-group-name"
		optionNameNodeName      = "node-name"
	)

	var (
		clusterName   string
		nodeGroupName string
		nodeName      string
	)

	cmd := &cobra.Command{
		Use:   "node",
		Short: "Delete Bee node",
		Long:  `Delete Bee node.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return fmt.Errorf("to be rethinked, subset of start cluster, probably not needed anymore")
		},
		PreRunE: c.deletePreRunE,
	}

	cmd.PersistentFlags().StringVar(&clusterName, optionNameClusterName, "bee", "cluster name")
	cmd.PersistentFlags().StringVar(&nodeGroupName, optionNameNodeGroupName, "bee", "node group name")
	cmd.PersistentFlags().StringVar(&nodeName, optionNameNodeName, "bee", "node name")

	return cmd
}

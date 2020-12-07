package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
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
			cluster := bee.NewCluster(clusterName, bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				KubeconfigPath:      c.config.GetString(optionNameStartKubeconfig),
				Namespace:           c.config.GetString(optionNameStartNamespace),
			})

			// node group
			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup(nodeGroupName, *ngOptions)
			ng := cluster.NodeGroup(nodeGroupName)

			return ng.DeleteNode(cmd.Context(), nodeName)
		},
		PreRunE: c.deletePreRunE,
	}

	cmd.PersistentFlags().StringVar(&clusterName, optionNameClusterName, "bee", "cluster name")
	cmd.PersistentFlags().StringVar(&nodeGroupName, optionNameNodeGroupName, "bee", "node group name")
	cmd.PersistentFlags().StringVar(&nodeName, optionNameNodeName, "bee", "node name")

	return cmd
}

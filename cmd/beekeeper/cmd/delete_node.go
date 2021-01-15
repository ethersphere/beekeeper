package cmd

import (
	"fmt"

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
			k8sClient, err := setK8SClient(c.config.GetString(optionNameKubeconfig), c.config.GetBool(optionNameInCluster))
			if err != nil {
				return fmt.Errorf("creating Kubernetes client: %w", err)
			}

			cluster := bee.NewCluster(clusterName, bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				K8SClient:           k8sClient,
				Namespace:           c.config.GetString(optionNameNamespace),
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

package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initStartNode() *cobra.Command {
	const (
		createdBy                  = "beekeeper"
		labelName                  = "bee"
		managedBy                  = "beekeeper"
		optionNameBootnodes        = "bootnodes"
		optionNameClusterName      = "cluster-name"
		optionNameNodeGroupName    = "node-group-name"
		optionNameNodeGroupVersion = "node-group-version"
		optionNameNodeName         = "node-name"
		optionNameStartStandalone  = "standalone"
	)

	var (
		bootnodes        string
		clusterName      string
		nodeGroupName    string
		nodeGroupVersion string
		nodeName         string
		standalone       bool
	)

	cmd := &cobra.Command{
		Use:   "node",
		Short: "Start Bee node",
		Long:  `Start Bee node.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cluster := bee.NewCluster(clusterName, bee.ClusterOptions{
				Annotations: map[string]string{
					"created-by":        createdBy,
					"beekeeper/version": beekeeper.Version,
				},
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				KubeconfigPath:      c.config.GetString(optionNameStartKubeconfig),
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": managedBy,
					"app.kubernetes.io/name":       labelName,
				},
				Namespace: c.config.GetString(optionNameStartNamespace),
			})

			// node group
			ngOptions := newDefaultNodeGroupOptions()
			ngOptions.Image = fmt.Sprintf("ethersphere/bee:%s", nodeGroupVersion)
			ngOptions.Labels = map[string]string{
				"app.kubernetes.io/component": nodeGroupName,
				"app.kubernetes.io/part-of":   nodeGroupName,
				"app.kubernetes.io/version":   nodeGroupVersion,
			}
			cluster.AddNodeGroup(nodeGroupName, ngOptions)
			ng := cluster.NodeGroup(nodeGroupName)

			nodeConfig := newBeeDefaultConfig()
			nodeConfig.Bootnodes = bootnodes
			nodeConfig.Standalone = standalone

			return ng.StartNode(cmd.Context(), bee.StartNodeOptions{
				Name:   nodeName,
				Config: nodeConfig,
			})
		},
		PreRunE: c.startPreRunE,
	}

	cmd.PersistentFlags().StringVar(&bootnodes, optionNameBootnodes, "", "bootnodes")
	cmd.PersistentFlags().StringVar(&clusterName, optionNameClusterName, "bee", "cluster name")
	cmd.PersistentFlags().StringVar(&nodeGroupName, optionNameNodeGroupName, "bee", "node group name")
	cmd.PersistentFlags().StringVar(&nodeGroupVersion, optionNameNodeGroupVersion, "latest", "node group version")
	cmd.PersistentFlags().StringVar(&nodeName, optionNameNodeName, "bee", "node name")
	cmd.PersistentFlags().BoolVarP(&standalone, optionNameStartStandalone, "s", false, "start a standalone node")

	return cmd
}

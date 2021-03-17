package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
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
			k8sClient, err := setK8SClient(c.config.GetString(optionNameKubeconfig), c.config.GetBool(optionNameInCluster))
			if err != nil {
				return fmt.Errorf("creating Kubernetes client: %w", err)
			}

			namespace := c.config.GetString(optionNameNamespace)
			cluster := bee.NewCluster(clusterName, bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				K8SClient:           k8sClient,
				Namespace:           namespace,
			})

			// node groups
			if additionalNodeCount > 0 {
				addNgName := "drone"
				addNgOptions := newDefaultNodeGroupOptions()
				cluster.AddNodeGroup(addNgName, *addNgOptions)
				addNg := cluster.NodeGroup(addNgName)

				for i := 0; i < additionalNodeCount; i++ {
					if err := addNg.DeleteNode(cmd.Context(), fmt.Sprintf("drone-%d", i)); err != nil {
						return fmt.Errorf("deleting drone-%d: %w", i, err)
					}
				}
			}

			ngName := "bee"
			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup(ngName, *ngOptions)
			ng := cluster.NodeGroup(ngName)

			for i := 0; i < nodeCount; i++ {
				if err := ng.DeleteNode(cmd.Context(), fmt.Sprintf("bee-%d", i)); err != nil {
					return fmt.Errorf("deleting bee-%d: %w", i, err)
				}
			}

			// bootnodes group
			bgName := "bootnodes"
			bgOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup(bgName, *bgOptions)
			bg := cluster.NodeGroup(bgName)
			for i := 0; i < bootnodeCount; i++ {
				if err := bg.DeleteNode(cmd.Context(), fmt.Sprintf("bootnode-%d", i)); err != nil {
					return fmt.Errorf("deleting bootnode-%d: %w", i, err)
				}
			}

			return
		},
		PreRunE: c.deletePreRunE,
	}

	cmd.PersistentFlags().StringVar(&clusterName, optionNameClusterName, "bee", "cluster name")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 1, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")

	return cmd
}

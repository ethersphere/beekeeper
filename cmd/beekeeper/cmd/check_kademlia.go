package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/kademlia"

	"github.com/spf13/cobra"
)

func (c *command) initCheckKademlia() *cobra.Command {
	const (
		optionNameDynamic       = "dynamic"
		optionNameClusterName   = "cluster-name"
		optionNameBootnodeCount = "bootnode-count"
		optionNameNodeCount     = "node-count"
	)

	var (
		dynamic       bool
		clusterName   string
		bootnodeCount int
		nodeCount     int
	)

	cmd := &cobra.Command{
		Use:   "kademlia",
		Short: "Checks Kademlia topology in the cluster",
		Long:  `Checks Kademlia topology in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cluster := bee.NewCluster(clusterName, bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				Namespace:           c.config.GetString(optionNameNamespace),
			})

			if bootnodeCount > 0 {
				// bootnodes group
				bgName := "bootnodes"
				bgOptions := newDefaultNodeGroupOptions()
				cluster.AddNodeGroup(bgName, *bgOptions)
				bg := cluster.NodeGroup(bgName)

				for i := 0; i < bootnodeCount; i++ {
					if err := bg.AddNode(fmt.Sprintf("bootnode-%d", i)); err != nil {
						return fmt.Errorf("adding bootnode-%d: %v", i, err)
					}
				}
			}

			// nodes group
			ngName := "nodes"
			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup(ngName, *ngOptions)
			ng := cluster.NodeGroup(ngName)

			for i := 0; i < nodeCount; i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i)); err != nil {
					return fmt.Errorf("adding bee-%d: %v", i, err)
				}
			}

			if dynamic {
				return kademlia.CheckDynamic(cmd.Context(), cluster)
			}

			return kademlia.Check(cmd.Context(), cluster)
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().BoolVar(&dynamic, optionNameDynamic, false, "check on dynamic cluster")
	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 0, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")

	return cmd
}

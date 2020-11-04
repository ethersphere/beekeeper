package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initStartCluster() *cobra.Command {
	const (
		createdBy                   = "beekeeper"
		labelName                   = "bee"
		managedBy                   = "beekeeper"
		optionNameClusterName       = "cluster-name"
		optionNameNodeGroupName     = "node-group-name"
		optionNameNodeGroup2Name    = "node-group2-name"
		optionNameNodeGroupVersion  = "node-group-version"
		optionNameNodeGroup2Version = "node-group2-version"
	)

	var (
		clusterName       string
		nodeGroupName     string
		nodeGroupVersion  string
		nodeGroup2Name    string
		nodeGroup2Version string
	)

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Start Bee cluster",
		Long:  `Start Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx := context.Background()

			cluster := bee.NewDynamicCluster(clusterName, bee.DynamicClusterOptions{
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
			ngOptions := defaultNodeGroupOptions
			ngOptions.Image = fmt.Sprintf("ethersphere/bee:%s", nodeGroupVersion)
			ngOptions.Labels = map[string]string{
				"app.kubernetes.io/component": nodeGroupName,
				"app.kubernetes.io/part-of":   nodeGroupName,
				"app.kubernetes.io/version":   nodeGroupVersion,
			}
			cluster.AddNodeGroup(nodeGroupName, ngOptions)
			ng := cluster.NodeGroup(nodeGroupName)

			// node group 2
			ng2Options := defaultNodeGroupOptions
			ng2Options.Image = fmt.Sprintf("ethersphere/bee:%s", nodeGroup2Version)
			ng2Options.Labels = map[string]string{
				"app.kubernetes.io/component": nodeGroup2Name,
				"app.kubernetes.io/part-of":   nodeGroup2Name,
				"app.kubernetes.io/version":   nodeGroup2Version,
			}
			cluster.AddNodeGroup(nodeGroup2Name, ng2Options)
			ng2 := cluster.NodeGroup(nodeGroup2Name)

			bn1Config := beeDefaultConfig
			bn1Config.Bootnodes = fmt.Sprintf(setup1.Bootnode, "bootnode-1", "beekeeper")
			if err := ng.NodeStart(ctx, bee.NodeStartOptions{
				Name:         "bootnode-0",
				Config:       bn1Config,
				ClefKey:      setup0.ClefKey,
				ClefPassword: setup0.ClefPassword,
				LibP2PKey:    setup0.LibP2PKey,
				SwarmKey:     setup0.SwarmKey,
			}); err != nil {
				return err
			}

			bn2Config := beeDefaultConfig
			bn2Config.Bootnodes = fmt.Sprintf(setup0.Bootnode, "bootnode-0", "beekeeper")
			if err := ng.NodeStart(ctx, bee.NodeStartOptions{
				Name:         "bootnode-1",
				Config:       bn2Config,
				ClefKey:      setup1.ClefKey,
				ClefPassword: setup1.ClefPassword,
				LibP2PKey:    setup1.LibP2PKey,
				SwarmKey:     setup1.SwarmKey,
			}); err != nil {
				return err
			}

			b1Config := beeDefaultConfig
			b1Config.Bootnodes = fmt.Sprintf(setup0.Bootnode, "bootnode-0", "beekeeper") + " " + fmt.Sprintf(setup1.Bootnode, "bootnode-1", "beekeeper")
			if err := ng2.NodeStart(ctx, bee.NodeStartOptions{
				Name:   "bee-0",
				Config: b1Config,
			}); err != nil {
				return err
			}

			b2Config := beeDefaultConfig
			b2Config.Bootnodes = fmt.Sprintf(setup0.Bootnode, "bootnode-0", "beekeeper") + " " + fmt.Sprintf(setup1.Bootnode, "bootnode-1", "beekeeper")
			if err := ng2.NodeStart(ctx, bee.NodeStartOptions{
				Name:   "bee-1",
				Config: b2Config,
			}); err != nil {
				return err
			}

			b3Config := beeDefaultConfig
			b3Config.Bootnodes = fmt.Sprintf(setup0.Bootnode, "bootnode-0", "beekeeper") + " " + fmt.Sprintf(setup1.Bootnode, "bootnode-1", "beekeeper")
			if err := ng2.NodeStart(ctx, bee.NodeStartOptions{
				Name:   "bee-2",
				Config: b3Config,
			}); err != nil {
				return err
			}

			return
		},
		PreRunE: c.startPreRunE,
	}

	cmd.PersistentFlags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.PersistentFlags().StringVar(&nodeGroupName, optionNameNodeGroupName, "bootnode", "node group name")
	cmd.PersistentFlags().StringVar(&nodeGroupVersion, optionNameNodeGroupVersion, "latest", "node group version")
	cmd.PersistentFlags().StringVar(&nodeGroup2Name, optionNameNodeGroup2Name, "bee", "node group name")
	cmd.PersistentFlags().StringVar(&nodeGroup2Version, optionNameNodeGroup2Version, "latest", "node group version")

	return cmd
}

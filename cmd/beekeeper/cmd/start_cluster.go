package cmd

import (
	"fmt"
	"strings"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initStartCluster() *cobra.Command {
	const (
		createdBy                = "beekeeper"
		labelName                = "bee"
		managedBy                = "beekeeper"
		optionNameClusterName    = "cluster-name"
		optionNameImage          = "bee-image"
		optionNameBootnodeCount  = "bootnode-count"
		optionNameNodeCount      = "node-count"
		optionNamePersistence    = "persistence"
		optionNameStorageClass   = "storage-class"
		optionNameStorageRequest = "storage-request"
	)

	var (
		clusterName    string
		image          string
		bootnodeCount  int
		nodeCount      int
		persistence    bool
		storageClass   string
		storageRequest string
	)

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Start Bee cluster",
		Long:  `Start Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			k8sClient, err := setK8SClient(c.config.GetString(optionNameKubeconfig), c.config.GetBool(optionNameInCluster))
			if err != nil {
				return fmt.Errorf("creating Kubernetes client: %v", err)
			}

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
				K8SClient:           k8sClient,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": managedBy,
					"app.kubernetes.io/name":       labelName,
				},
				Namespace: c.config.GetString(optionNameNamespace),
			})

			// bootnodes group
			bgName := "bootnodes"
			bgOptions := newDefaultNodeGroupOptions()
			bgOptions.Image = image
			bgOptions.Labels = map[string]string{
				"app.kubernetes.io/component": "bootnode",
				"app.kubernetes.io/part-of":   bgName,
				"app.kubernetes.io/version":   strings.Split(image, ":")[1],
			}
			bgOptions.PersistenceEnabled = persistence
			bgOptions.PersistenceStorageClass = storageClass
			bgOptions.PersistanceStorageRequest = storageRequest
			cluster.AddNodeGroup(bgName, *bgOptions)
			bg := cluster.NodeGroup(bgName)

			bSetup := setupBootnodes(bootnodeCount, c.config.GetString(optionNameNamespace))
			for i := 0; i < bootnodeCount; i++ {
				bConfig := newBeeDefaultConfig()
				bConfig.Bootnodes = bSetup[i].Bootnodes
				if err := bg.StartNode(cmd.Context(), bee.StartNodeOptions{
					Name:         fmt.Sprintf("bootnode-%d", i),
					Config:       *bConfig,
					ClefKey:      bSetup[i].ClefKey,
					ClefPassword: bSetup[i].ClefPassword,
					LibP2PKey:    bSetup[i].LibP2PKey,
					SwarmKey:     bSetup[i].SwarmKey,
				}); err != nil {
					return fmt.Errorf("starting bootnode-%d: %s", i, err)
				}
			}

			// nodes group
			ngName := "nodes"
			ngOptions := newDefaultNodeGroupOptions()
			ngOptions.Image = image
			ngOptions.Labels = map[string]string{
				"app.kubernetes.io/component": "node",
				"app.kubernetes.io/part-of":   ngName,
				"app.kubernetes.io/version":   strings.Split(image, ":")[1],
			}
			ngOptions.PersistenceEnabled = persistence
			ngOptions.PersistenceStorageClass = storageClass
			ngOptions.PersistanceStorageRequest = storageRequest
			cluster.AddNodeGroup(ngName, *ngOptions)
			ng := cluster.NodeGroup(ngName)

			nConfig := newBeeDefaultConfig()
			nConfig.Bootnodes = setupBootnodesDNS(bootnodeCount, c.config.GetString(optionNameNamespace))
			for i := 0; i < nodeCount; i++ {
				if err := ng.StartNode(cmd.Context(), bee.StartNodeOptions{
					Name:   fmt.Sprintf("bee-%d", i),
					Config: *nConfig,
				}); err != nil {
					return fmt.Errorf("starting bee-%d: %s", i, err)
				}
			}

			return
		},
		PreRunE: c.startPreRunE,
	}

	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 1, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")

	return cmd
}

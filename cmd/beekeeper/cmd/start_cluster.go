package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initStartCluster() *cobra.Command {
	const (
		createdBy                     = "beekeeper"
		labelName                     = "bee"
		managedBy                     = "beekeeper"
		optionNameClusterName         = "cluster-name"
		optionNameImage               = "bee-image"
		optionNameAdditionalImage     = "additional-bee-image"
		optionNameBootnodeCount       = "bootnode-count"
		optionNameNodeCount           = "node-count"
		optionNameAdditionalNodeCount = "additional-node-count"
		optionNamePersistence         = "persistence"
		optionNameStorageClass        = "storage-class"
		optionNameStorageRequest      = "storage-request"
	)

	var (
		clusterName         string
		image               string
		additionalImage     string
		bootnodeCount       int
		nodeCount           int
		additionalNodeCount int
		persistence         bool
		storageClass        string
		storageRequest      string
	)

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Start Bee cluster",
		Long:  `Start Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			k8sClient, err := setK8SClient(c.config.GetString(optionNameKubeconfig), c.config.GetBool(optionNameInCluster))
			if err != nil {
				return fmt.Errorf("creating Kubernetes client: %w", err)
			}

			namespace := c.config.GetString(optionNameNamespace)
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
				Namespace: namespace,
			})

			// bootnodes group
			bgName := "bootnode"
			bCtx, bCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer bCancel()
			if err := startBootNodeGroup(bCtx, cluster, bootnodeCount, nodeCount, bgName, namespace, image, storageClass, storageRequest, persistence); err != nil {
				return fmt.Errorf("starting bootnode group %s: %w", bgName, err)
			}

			// node groups
			ngName := "bee"
			nCtx, nCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer nCancel()
			if err := startNodeGroup(nCtx, cluster, bootnodeCount, nodeCount, ngName, namespace, image, storageClass, storageRequest, persistence); err != nil {
				return fmt.Errorf("starting node group %s: %w", ngName, err)
			}

			if additionalNodeCount > 0 {
				addNgName := "drone"
				addNCtx, addNCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer addNCancel()
				if err := startNodeGroup(addNCtx, cluster, bootnodeCount, additionalNodeCount, addNgName, namespace, additionalImage, storageClass, storageRequest, persistence); err != nil {
					return fmt.Errorf("starting node group %s: %w", addNgName, err)
				}
			}

			return
		},
		PreRunE: c.startPreRunE,
	}

	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.Flags().StringVar(&additionalImage, optionNameAdditionalImage, "ethersphere/bee-netem:latest", "Bee Docker image in additional node group")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 1, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")

	return cmd
}

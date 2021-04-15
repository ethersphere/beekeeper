package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/check/pingpong"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initCheckPing() *cobra.Command {
	const (
		optionNameStartCluster        = "start-cluster"
		optionNameDynamic             = "dynamic"
		optionNameClusterName         = "cluster-name"
		optionNameBootnodeCount       = "bootnode-count"
		optionNameNodeCount           = "node-count"
		optionNameAdditionalNodeCount = "additional-node-count"
		optionNameImage               = "bee-image"
		optionNameAdditionalImage     = "additional-bee-image"
		optionNameSeed                = "seed"
		optionNamePersistence         = "persistence"
		optionNameStorageClass        = "storage-class"
		optionNameStorageRequest      = "storage-request"
	)

	var (
		startCluster        bool
		dynamic             bool
		clusterName         string
		bootnodeCount       int
		nodeCount           int
		additionalNodeCount int
		image               string
		additionalImage     string
		persistence         bool
		storageClass        string
		storageRequest      string
	)

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Executes ping from all nodes to all other nodes in the cluster",
		Long: `Executes ping from all nodes to all other nodes in the cluster,
and prints round-trip time (RTT) of each ping.`,
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
				DisableNamespace:    disableNamespace,
			})

			if startCluster {
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
			} else {
				// bootnodes group
				if bootnodeCount > 0 {
					bgName := "bootnode"
					if err := addBootNodeGroup(cluster, bootnodeCount, nodeCount, bgName, namespace, image, storageClass, storageRequest, persistence); err != nil {
						return fmt.Errorf("adding bootnode group %s: %w", bgName, err)
					}
				}

				// bee nodes group
				ngName := "bee"
				if err := addNodeGroup(cluster, bootnodeCount, nodeCount, ngName, namespace, image, storageClass, storageRequest, persistence); err != nil {
					return fmt.Errorf("adding node group %s: %w", ngName, err)
				}
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}
			buffer := 12

			checkCtx, checkCancel := context.WithTimeout(cmd.Context(), 15*time.Minute)
			defer checkCancel()

			checkPing := pingpong.NewPing()
			checkOptions := check.Options{
				MetricsEnabled: c.config.GetBool(optionNamePushMetrics),
				MetricsPusher:  push.New(c.config.GetString(optionNamePushGateway), namespace),
			}

			return check.RunConcurrently(checkCtx, cluster, checkPing, checkOptions, []check.Stage{}, buffer, seed)
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")
	cmd.Flags().BoolVar(&dynamic, optionNameDynamic, false, "check on dynamic cluster")
	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 0, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.Flags().StringVar(&additionalImage, optionNameAdditionalImage, "ethersphere/bee-netem:latest", "Bee Docker image in additional node group")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")

	return cmd
}

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
		optionNameStartCluster             = "start-cluster"
		optionNameDynamic                  = "dynamic"
		optionNameClusterName              = "cluster-name"
		optionNameImagePullSecrets         = "image-pull-secrets"
		optionNameBootnodeCount            = "bootnode-count"
		optionNameNodeCount                = "node-count"
		optionNameImage                    = "bee-image"
		optionNameFullNode                 = "full-node"
		optionNamePersistence              = "persistence"
		optionNameStorageClass             = "storage-class"
		optionNameStorageRequest           = "storage-request"
		optionNameAdditionalNodeCount      = "additional-node-count"
		optionNameAdditionalImage          = "additional-bee-image"
		optionNameAdditionalFullNode       = "additional-full-node"
		optionNameAdditionalPersistence    = "additional-persistence"
		optionNameAdditionalStorageClass   = "additional-storage-class"
		optionNameAdditionalStorageRequest = "additional-storage-request"
		optionNameSeed                     = "seed"
	)

	var (
		startCluster             bool
		dynamic                  bool
		clusterName              string
		imagePullSecrets         []string
		bootnodeCount            int
		nodeCount                int
		image                    string
		fullNode                 bool
		persistence              bool
		storageClass             string
		storageRequest           string
		additionalNodeCount      int
		additionalImage          string
		additionalFullNode       bool
		additionalPersistence    bool
		additionalStorageClass   string
		additionalStorageRequest string
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

			cicd := newCICDOptions(clefSignerEnable, cacheCapacity, paymentEarly, paymentThreshold, paymentTolerance, swapEnable, swapEndpoint, swapFactoryAddress, swapInitialDeposit, nodeSelector, ingressClass)

			if startCluster {
				// bootnodes group
				bgName := "bootnode"
				bCtx, bCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer bCancel()
				if err := startBootNodeGroup(bCtx, cluster, bootnodeCount, nodeCount, bgName, namespace, image, storageClass, storageRequest, imagePullSecrets, persistence, cicd); err != nil {
					return fmt.Errorf("starting bootnode group %s: %w", bgName, err)
				}

				// node groups
				ngName := "bee"
				nCtx, nCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer nCancel()
				if err := startNodeGroup(nCtx, cluster, bootnodeCount, nodeCount, ngName, namespace, image, storageClass, storageRequest, imagePullSecrets, persistence, fullNode, cicd); err != nil {
					return fmt.Errorf("starting node group %s: %w", ngName, err)
				}

				if additionalNodeCount > 0 {
					addNgName := "drone"
					addNCtx, addNCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
					defer addNCancel()
					if err := startNodeGroup(addNCtx, cluster, bootnodeCount, additionalNodeCount, addNgName, namespace, additionalImage, additionalStorageClass, additionalStorageRequest, imagePullSecrets, additionalPersistence, additionalFullNode, cicd); err != nil {
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

				// node groups
				ngName := "bee"
				if err := addNodeGroup(cluster, bootnodeCount, nodeCount, ngName, namespace, image, storageClass, storageRequest, persistence); err != nil {
					return fmt.Errorf("adding node group %s: %w", ngName, err)
				}

				if additionalNodeCount > 0 {
					addNgName := "drone"
					if err := addNodeGroup(cluster, bootnodeCount, additionalNodeCount, addNgName, namespace, additionalImage, additionalStorageClass, additionalStorageRequest, additionalPersistence); err != nil {
						return fmt.Errorf("starting node group %s: %w", addNgName, err)
					}
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

	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")
	cmd.Flags().BoolVar(&dynamic, optionNameDynamic, false, "check on dynamic cluster")
	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().StringArrayVar(&imagePullSecrets, optionNameImagePullSecrets, []string{"regcred"}, "image pull secrets")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 0, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.PersistentFlags().BoolVar(&fullNode, optionNameFullNode, true, "start node in full mode")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, true, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")
	cmd.Flags().StringVar(&additionalImage, optionNameAdditionalImage, "ethersphere/bee:latest", "Bee Docker image in additional node group")
	cmd.PersistentFlags().BoolVar(&additionalFullNode, optionNameAdditionalFullNode, false, "start node in full mode")
	cmd.PersistentFlags().BoolVar(&additionalPersistence, optionNameAdditionalPersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&additionalStorageClass, optionNameAdditionalStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&additionalStorageRequest, optionNameAdditionalStorageRequest, "34Gi", "storage request")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")

	return cmd
}

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/stress"
	"github.com/ethersphere/beekeeper/pkg/stress/upload"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initStressUpload() *cobra.Command {
	const (
		optionNameStartCluster          = "start-cluster"
		optionNameDynamic               = "dynamic"
		optionNameClusterName           = "cluster-name"
		optionNameBootnodeCount         = "bootnode-count"
		optionNameNodeCount             = "node-count"
		optionNameImage                 = "bee-image"
		optionNameAdditionalImage       = "additional-bee-image"
		optionNameAdditionalNodeCount   = "additional-node-count"
		optionNameSeed                  = "seed"
		optionNamePersistence           = "persistence"
		optionNameStorageClass          = "storage-class"
		optionNameStorageRequest        = "storage-request"
		optionNameUploadNodesPercentage = "upload-nodes-percentage"
		optionNameTimeout               = "timeout"
		optionNameFileSize              = "file-size"
		optionNameRetries               = "retries"
		optionNameRetryDelay            = "retry-delay"
	)

	var (
		startCluster          bool
		dynamic               bool
		clusterName           string
		bootnodeCount         int
		nodeCount             int
		additionalNodeCount   int
		image                 string
		additionalImage       string
		persistence           bool
		storageClass          string
		storageRequest        string
		uploadNodesPercentage int
	)

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Uploads data to all nodes in the cluster",
		Long:  `Uploads data to all nodes in the cluster to ensure that the GC process is activated.`,
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

				// nodes group
				ngName := "bee"
				if err := addNodeGroup(cluster, bootnodeCount, nodeCount, ngName, namespace, image, storageClass, storageRequest, persistence); err != nil {
					return fmt.Errorf("adding node group %s: %w", ngName, err)
				}

				if additionalNodeCount > 0 {
					addNgName := "drone"
					if err := addNodeGroup(cluster, bootnodeCount, additionalNodeCount, addNgName, namespace, additionalImage, storageClass, storageRequest, persistence); err != nil {
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

			if uploadNodesPercentage < 0 || uploadNodesPercentage > 100 {
				return fmt.Errorf("upload-nodes-percentage must be number between 0 and 100")
			}

			stressUpload := upload.NewUpload()
			stressOptions := stress.Options{
				FileSize:              round(c.config.GetFloat64(optionNameFileSize) * 1024 * 1024),
				MetricsEnabled:        c.config.GetBool(optionNamePushMetrics),
				MetricsPusher:         push.New(c.config.GetString(optionNamePushGateway), namespace),
				Retries:               c.config.GetInt(optionNameRetries),
				RetryDelay:            c.config.GetDuration(optionNameRetryDelay),
				Seed:                  seed,
				Timeout:               c.config.GetDuration(optionNameTimeout),
				UploadNodesPercentage: uploadNodesPercentage,
			}

			dynamicStages := []stress.Stage{}
			if dynamic {
				dynamicStages = stressStages
			}

			return stress.RunConcurrently(cmd.Context(), cluster, stressUpload, stressOptions, dynamicStages, buffer, seed)
		},
		PreRunE: c.stressPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")
	cmd.Flags().BoolVar(&dynamic, optionNameDynamic, false, "stress on dynamic cluster")
	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 0, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.Flags().StringVar(&additionalImage, optionNameAdditionalImage, "ethersphere/bee-netem:latest", "Bee Docker image in additional node group")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")
	cmd.PersistentFlags().IntVar(&uploadNodesPercentage, optionNameUploadNodesPercentage, 50, "percentage of nodes to upload to")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "how long to upload files on each node")
	cmd.Flags().Float64(optionNameFileSize, 1, "file size in MB")
	cmd.Flags().Int(optionNameRetries, 5, "number of reties on problems")
	cmd.Flags().Duration(optionNameRetryDelay, time.Second, "retry delay duration")

	return cmd
}

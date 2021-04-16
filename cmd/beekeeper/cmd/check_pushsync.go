package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPushSync() *cobra.Command {
	const (
		optionNameUploadNodeCount          = "upload-node-count"
		optionNameChunksPerNode            = "chunks-per-node"
		optionNameFilesPerNode             = "files-per-node"
		optionNameSeed                     = "seed"
		optionNameConcurrent               = "concurrent"
		optionNameUploadChunks             = "upload-chunks"
		optionNameUploadFiles              = "upload-files"
		optionNameFileSize                 = "file-size"
		optionNameRetries                  = "retries"
		optionNameRetryDelay               = "retry-delay"
		optionNameStartCluster             = "start-cluster"
		optionNameClusterName              = "cluster-name"
		optionNameBootnodeCount            = "bootnode-count"
		optionNameNodeCount                = "node-count"
		optionNameImage                    = "bee-image"
		optionNamePersistence              = "persistence"
		optionNameStorageClass             = "storage-class"
		optionNameStorageRequest           = "storage-request"
		optionNameFullNode                 = "full-node"
		optionNameAdditionalNodeCount      = "additional-node-count"
		optionNameAdditionalImage          = "additional-bee-image"
		optionNameAdditionalFullNode       = "additional-full-node"
		optionNameAdditionalPersistence    = "additional-persistence"
		optionNameAdditionalStorageClass   = "additional-storage-class"
		optionNameAdditionalStorageRequest = "additional-storage-request"
		optionNameImagePullSecrets         = "image-pull-secrets"
	)

	var (
		imagePullSecrets         []string
		startCluster             bool
		clusterName              string
		bootnodeCount            int
		nodeCount                int
		image                    string
		persistence              bool
		storageClass             string
		storageRequest           string
		concurrent               bool
		uploadChunks             bool
		uploadFiles              bool
		fullNode                 bool
		additionalNodeCount      int
		additionalImage          string
		additionalFullNode       bool
		additionalPersistence    bool
		additionalStorageClass   string
		additionalStorageRequest string
	)

	cmd := &cobra.Command{
		Use:   "pushsync",
		Short: "Checks pushsync ability of the cluster",
		Long: `Checks pushsync ability of the cluster.
It uploads given number of chunks to given number of nodes, 
and checks if chunks are synced to their closest nodes.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if c.config.GetInt(optionNameUploadNodeCount) > c.config.GetInt(optionNameNodeCount) {
				return errors.New("bad parameters: upload-node-count must be less or equal to node-count")
			}

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
				if err := startBootNodeGroup(bCtx, cluster, bootnodeCount, nodeCount, bgName, namespace, image, storageClass, storageRequest, imagePullSecrets, persistence); err != nil {
					return fmt.Errorf("starting bootnode group %s: %w", bgName, err)
				}

				// node groups
				ngName := "bee"
				nCtx, nCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer nCancel()
				if err := startNodeGroup(nCtx, cluster, bootnodeCount, nodeCount, ngName, namespace, image, storageClass, storageRequest, imagePullSecrets, persistence, fullNode); err != nil {
					return fmt.Errorf("starting node group %s: %w", ngName, err)
				}

				if additionalNodeCount > 0 {
					addNgName := "drone"
					addNCtx, addNCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
					defer addNCancel()
					if err := startNodeGroup(addNCtx, cluster, bootnodeCount, additionalNodeCount, addNgName, namespace, additionalImage, additionalStorageClass, additionalStorageRequest, imagePullSecrets, additionalPersistence, additionalFullNode); err != nil {
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

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			if concurrent {
				return pushsync.CheckConcurrent(cluster, pushsync.Options{
					NodeGroup:       "bee",
					UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
					ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
					Seed:            seed,
				})
			}

			if uploadChunks {
				return pushsync.CheckChunks(cluster, pushsync.Options{
					NodeGroup:       "bee",
					UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
					ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
					RetryDelay:      c.config.GetDuration(optionNameRetryDelay),
					Seed:            seed,
				})
			}

			if uploadFiles {
				fileSize := round(c.config.GetFloat64(optionNameFileSize) * 1024 * 1024)
				retryDelayDuration := c.config.GetDuration(optionNameRetryDelay)

				return pushsync.CheckFiles(cluster, pushsync.Options{
					NodeGroup:       "bee",
					UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
					ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
					FilesPerNode:    c.config.GetInt(optionNameFilesPerNode),
					FileSize:        fileSize,
					Retries:         c.config.GetInt(optionNameRetries),
					RetryDelay:      retryDelayDuration,
					Seed:            seed,
				})
			}

			retryDelayDuration := c.config.GetDuration(optionNameRetryDelay)
			return pushsync.Check(cluster, pushsync.Options{
				NodeGroup:       "bee",
				UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
				ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
				Retries:         c.config.GetInt(optionNameRetries),
				RetryDelay:      retryDelayDuration,
				Seed:            seed,
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload to")
	cmd.Flags().IntP(optionNameChunksPerNode, "p", 1, "number of data to upload per node")
	cmd.Flags().IntP(optionNameFilesPerNode, "f", 1, "number of files to upload per node")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().BoolVar(&concurrent, optionNameConcurrent, false, "upload concurrently")
	cmd.Flags().BoolVar(&uploadChunks, optionNameUploadChunks, false, "upload chunks")
	cmd.Flags().BoolVar(&uploadFiles, optionNameUploadFiles, false, "upload files")
	cmd.Flags().Float64(optionNameFileSize, 1, "file size in MB")
	cmd.Flags().Int(optionNameRetries, 5, "number of reties on problems")
	cmd.Flags().Duration(optionNameRetryDelay, time.Second, "retry delay duration")
	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")
	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 0, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")
	cmd.PersistentFlags().BoolVar(&fullNode, optionNameFullNode, true, "start node in full mode")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")
	cmd.Flags().StringVar(&additionalImage, optionNameAdditionalImage, "anatollupacescu/light-nodes:latest", "Bee Docker image in additional node group")
	cmd.PersistentFlags().BoolVar(&additionalFullNode, optionNameAdditionalFullNode, false, "start node in full mode")
	cmd.PersistentFlags().BoolVar(&additionalPersistence, optionNameAdditionalPersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&additionalStorageClass, optionNameAdditionalStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&additionalStorageRequest, optionNameAdditionalStorageRequest, "34Gi", "storage request")
	cmd.Flags().StringArrayVar(&imagePullSecrets, optionNameImagePullSecrets, []string{"regcred"}, "image pull secrets")

	return cmd
}

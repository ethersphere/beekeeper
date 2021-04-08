package cmd

import (
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
		optionNameUploadNodeCount = "upload-node-count"
		optionNameChunksPerNode   = "chunks-per-node"
		optionNameFilesPerNode    = "files-per-node"
		optionNameSeed            = "seed"
		optionNameConcurrent      = "concurrent"
		optionNameUploadChunks    = "upload-chunks"
		optionNameUploadFiles     = "upload-files"
		optionNameFileSize        = "file-size"
		optionNameRetries         = "retries"
		optionNameRetryDelay      = "retry-delay"
	)

	var (
		concurrent   bool
		uploadChunks bool
		uploadFiles  bool
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

			cluster := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", *ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i), bee.NodeOptions{}); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
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
					NodeGroup:       "nodes",
					UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
					ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
					Seed:            seed,
					PostageAmount:   c.config.GetInt64(optionNamePostageAmount),
					PostageWait:     c.config.GetDuration(optionNamePostageBatchhWait),
				})
			}

			if uploadChunks {
				return pushsync.CheckChunks(cluster, pushsync.Options{
					NodeGroup:       "nodes",
					UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
					ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
					RetryDelay:      c.config.GetDuration(optionNameRetryDelay),
					Seed:            seed,
					PostageAmount:   c.config.GetInt64(optionNamePostageAmount),
					PostageWait:     c.config.GetDuration(optionNamePostageBatchhWait),
				})
			}

			if uploadFiles {
				fileSize := round(c.config.GetFloat64(optionNameFileSize) * 1024 * 1024)
				retryDelayDuration := c.config.GetDuration(optionNameRetryDelay)

				return pushsync.CheckFiles(cluster, pushsync.Options{
					NodeGroup:       "nodes",
					UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
					ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
					FilesPerNode:    c.config.GetInt(optionNameFilesPerNode),
					FileSize:        fileSize,
					Retries:         c.config.GetInt(optionNameRetries),
					RetryDelay:      retryDelayDuration,
					Seed:            seed,
					PostageAmount:   c.config.GetInt64(optionNamePostageAmount),
					PostageWait:     c.config.GetDuration(optionNamePostageBatchhWait),
				})
			}

			retryDelayDuration := c.config.GetDuration(optionNameRetryDelay)
			return pushsync.Check(cluster, pushsync.Options{
				NodeGroup:       "nodes",
				UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
				ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
				Retries:         c.config.GetInt(optionNameRetries),
				RetryDelay:      retryDelayDuration,
				Seed:            seed,
				PostageAmount:   c.config.GetInt64(optionNamePostageAmount),
				PostageWait:     c.config.GetDuration(optionNamePostageBatchhWait),
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

	return cmd
}

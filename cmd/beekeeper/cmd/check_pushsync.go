package cmd

import (
	"errors"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPushSync() *cobra.Command {
	const (
		optionNameUploadNodeCount = "upload-node-count"
		optionNameChunksPerNode   = "chunks-per-node"
		optionNameSeed            = "seed"
		optionNameConcurrent      = "concurrent"
	)
	var concurrent bool

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

			cluster, err := bee.NewCluster(bee.ClusterOptions{
				APIScheme:               c.config.GetString(optionNameAPIScheme),
				APIHostnamePattern:      c.config.GetString(optionNameAPIHostnamePattern),
				APIDomain:               c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:          insecureTLSAPI,
				DebugAPIScheme:          c.config.GetString(optionNameDebugAPIScheme),
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS:     insecureTLSDebugAPI,
				Namespace:               c.config.GetString(optionNameNamespace),
				Size:                    c.config.GetInt(optionNameNodeCount),
			})
			if err != nil {
				return err
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			if concurrent {
				return pushsync.CheckConcurrent(cluster, pushsync.Options{
					UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
					ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
					Seed:            seed,
				})
			}

			return pushsync.Check2(cluster, pushsync.Options{
				UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
				ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
				Seed:            seed,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload chunks to")
	cmd.Flags().IntP(optionNameChunksPerNode, "p", 1, "number of chunks to upload per node")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().BoolVar(&concurrent, optionNameConcurrent, false, "upload chunks concurrently")

	return cmd
}

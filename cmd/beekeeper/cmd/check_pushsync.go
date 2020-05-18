package cmd

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPushSync() *cobra.Command {
	const (
		optionNameChunksPerNode   = "chunks-per-node"
		optionNameSeed            = "seed"
		optionNameUploadNodeCount = "upload-node-count"
	)

	cmd := &cobra.Command{
		Use:   "pushsync",
		Short: "Checks push sync",
		Long:  `Checks push sync`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if c.config.GetInt(optionNameUploadNodeCount) > c.config.GetInt(optionNameNodeCount) {
				return errors.New("upload-node-count must be less or equal to node-count")
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}
			fmt.Printf("seed: %d\n", seed)

			cluster, err := bee.NewCluster(bee.ClusterOptions{
				APIHostnamePattern:      c.config.GetString(optionNameAPIHostnamePattern),
				APIDomain:               c.config.GetString(optionNameAPIDomain),
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				Namespace:               c.config.GetString(optionNameNamespace),
				Size:                    c.config.GetInt(optionNameNodeCount),
			})
			if err != nil {
				return err
			}

			return pushsync.Check(cluster, pushsync.Options{
				ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
				Rand:            rand.New(rand.NewSource(seed)),
				UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameChunksPerNode, "p", 1, "number of chunks to upload per node")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload chunks to")

	return cmd
}

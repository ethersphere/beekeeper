package cmd

import (
	"errors"

	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPushSync() *cobra.Command {
	const (
		optionNameChunksPerNode   = "chunks-per-node"
		optionNameRandomSeed      = "random-seed"
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
			return check.PushSync(check.PushSyncOptions{
				APIHostnamePattern:      c.config.GetString(optionNameAPIHostnamePattern),
				APIDomain:               c.config.GetString(optionNameAPIDomain),
				ChunksPerNode:           c.config.GetInt(optionNameChunksPerNode),
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				DisableNamespace:        disableNamespace,
				Namespace:               c.config.GetString(optionNameNamespace),
				NodeCount:               c.config.GetInt(optionNameNodeCount),
				RandomSeed:              c.config.GetBool(optionNameRandomSeed),
				Seed:                    c.config.GetInt64(optionNameSeed),
				UploadNodeCount:         c.config.GetInt(optionNameUploadNodeCount),
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameChunksPerNode, "p", 1, "number of chunks to upload per node")
	cmd.Flags().BoolP(optionNameRandomSeed, "r", true, "random seed")
	cmd.Flags().Int64P(optionNameSeed, "s", 1, "seed")
	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload chunks to")

	return cmd
}

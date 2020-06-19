package cmd

import (
	"errors"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/fileretrieval"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckFileRetrieval() *cobra.Command {
	const (
		optionNameUploadNodeCount = "upload-node-count"
		optionNameFilesPerNode    = "files-per-node"
		optionNameFileName        = "file-name"
		optionNameFileSize        = "file-size"
		optionNameSeed            = "seed"
	)

	cmd := &cobra.Command{
		Use:   "fileretrieval",
		Short: "Checks file retrieval ability of the cluster",
		Long: `Checks file retrieval ability of the cluster.
It uploads given number of files to given number of nodes, 
and attempts retrieval of those files from the last node in the cluster.`,
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

			return fileretrieval.Check(cluster, fileretrieval.Options{
				UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
				FilesPerNode:    c.config.GetInt(optionNameFilesPerNode),
				FileName:        c.config.GetString(optionNameFileName),
				FileSize:        c.config.GetInt64(optionNameFileSize),
				Seed:            seed,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload files to")
	cmd.Flags().IntP(optionNameFilesPerNode, "p", 1, "number of files to upload per node")
	cmd.Flags().String(optionNameFileName, "file", "file name template")
	cmd.Flags().Int64(optionNameFileSize, 1048576, "file size in bytes")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")

	return cmd
}

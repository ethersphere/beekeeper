package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/manifest"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/spf13/cobra"
)

func (c *command) initCheckManifest() *cobra.Command {
	const (
		optionNameFilesInCollection = "files-in-collection"
		optionMaxPathnameLength     = "maximum-pathname-length"
		optionNameSeed              = "seed"
	)

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Checks manifest functionality on the cluster",
		Long: `Checks manifest functionality on the cluster.
It uploads given number of files archived in a collection to the first node in the cluster, 
and attempts retrieval of those files from the last node in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cluster, err := bee.NewCluster(bee.ClusterOptions{
				APIScheme:               c.config.GetString(optionNameAPIScheme),
				APIHostnamePattern:      c.config.GetString(optionNameAPIHostnamePattern),
				APIDomain:               c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:          insecureTLSAPI,
				DebugAPIScheme:          c.config.GetString(optionNameDebugAPIScheme),
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS:     insecureTLSDebugAPI,
				DisableNamespace:        disableNamespace,
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

			return manifest.Check(cluster, manifest.Options{
				FilesInCollection: c.config.GetInt(optionNameFilesInCollection),
				MaxPathnameLength: c.config.GetInt32(optionMaxPathnameLength),
				Seed:              seed,
			})

		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int(optionNameFilesInCollection, 10, "number of files to upload in single collection")
	cmd.Flags().Int32(optionMaxPathnameLength, 64, "maximum pathname length for files")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")

	return cmd
}

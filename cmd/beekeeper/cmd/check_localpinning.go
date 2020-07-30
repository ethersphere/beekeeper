package cmd

import (
	"errors"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/localpinning"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckLocalPinning() *cobra.Command {
	const (
		optionNameDBCapacity         = "db-capacity"
		optionNameFileName           = "file-name"
		optionNameLargeFileCount     = "large-file-count"
		optionNameLargeFileDiskRatio = "large-file-disk-ratio"
		optionNameSeed               = "seed"
		optionNameSmallFileDiskRatio = "small-file-disk-ratio"
	)

	cmd := &cobra.Command{
		Use:   "localpinning",
		Short: "Checks local pinning ability of the cluster.",
		Long: `Checks local pinning ability of the cluster.
It first uploads small reference file to a random node and tries downloading it from another node.
If everything goes well with the reference file, it uploads new small file to another random node and pins it.
It uploads large file to the same node, and tries downloading small file from other random node. This step can be repeated multiple times (large-file-count).
If everything goes well, it unpins the file.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if c.config.GetFloat64(optionNameSmallFileDiskRatio) >= c.config.GetFloat64(optionNameLargeFileDiskRatio) || c.config.GetFloat64(optionNameSmallFileDiskRatio) >= 1 {
				return errors.New("bad parameters: small-file-disk-ratio must be less than large-file-disk-ratio, and less than 1")
			}
			if c.config.GetInt64(optionNameDBCapacity) <= 0 || c.config.GetInt(optionNameLargeFileCount) <= 0 || c.config.GetFloat64(optionNameLargeFileDiskRatio) <= 0 || c.config.GetFloat64(optionNameSmallFileDiskRatio) <= 0 {
				return errors.New("bad parameters: db-capacity, large-file-count, large-file-disk-ratio and small-file-disk-ratio must be greater than 0")
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

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			smallFileSize := int64(c.config.GetFloat64(optionNameDBCapacity) * c.config.GetFloat64(optionNameSmallFileDiskRatio))
			largeFileSize := int64(c.config.GetFloat64(optionNameDBCapacity) * c.config.GetFloat64(optionNameLargeFileDiskRatio))

			return localpinning.Check(cluster, localpinning.Options{
				DBCapacity:     c.config.GetInt64(optionNameDBCapacity),
				FileName:       c.config.GetString(optionNameFileName),
				LargeFileCount: c.config.GetInt(optionNameLargeFileCount),
				LargeFileSize:  largeFileSize,
				Seed:           seed,
				SmallFileSize:  smallFileSize,
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Float64(optionNameDBCapacity, 500, "DB capacity in chunks")
	cmd.Flags().String(optionNameFileName, "file", "file name template")
	cmd.Flags().Int(optionNameLargeFileCount, 1, "number of large files to be uploaded")
	cmd.Flags().Float64(optionNameLargeFileDiskRatio, 0.1, "large-file-size = db-capacity * large-file-disk-ratio")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	cmd.Flags().Float64(optionNameSmallFileDiskRatio, 0.01, "small-file-size = db-capacity * small-file-disk-ratio")

	return cmd
}

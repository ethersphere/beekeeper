package cmd

import (
	"errors"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/fileretrievaldynamic"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initCheckFileRetrievalDynamic() *cobra.Command {
	const (
		optionNameDownloadNodeCount = "download-node-count"
		optionNameFileName          = "file-name"
		optionNameFileSize          = "file-size"
		optionNameNewNodeCount      = "new-node-count"
		optionNameSeed              = "seed"
		optionNameStopNodeCount     = "stop-node-count"
	)

	cmd := &cobra.Command{
		Use:   "fileretrievaldynamic",
		Short: "Checks file retrieval ability of the dynamic cluster",
		Long: `Checks file retrieval ability of the dynamic cluster.
It uploads given number of files to given number of nodes, 
and attempts retrieval of those files from the last node in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if c.config.GetInt(optionNameDownloadNodeCount) <= 0 {
				return errors.New("bad parameters: download-node-count must be greater than 0")
			}
			if c.config.GetInt(optionNameDownloadNodeCount) >= c.config.GetInt(optionNameNodeCount)-c.config.GetInt(optionNameStopNodeCount) {
				return errors.New("bad parameters: download-node-count must be less than node-count - stop-node-count")
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
				DisableNamespace:        disableNamespace,
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

			fileSize := round(c.config.GetFloat64(optionNameFileSize) * 1024 * 1024)

			return fileretrievaldynamic.Check(cluster, fileretrievaldynamic.Options{
				DownloadNodeCount: c.config.GetInt(optionNameDownloadNodeCount),
				FileName:          c.config.GetString(optionNameFileName),
				FileSize:          fileSize,
				NewNodeCount:      c.config.GetInt(optionNameNewNodeCount),
				Seed:              seed,
				StopNodeCount:     c.config.GetInt(optionNameStopNodeCount),
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameDownloadNodeCount, "d", 1, "number of nodes to download files from")
	cmd.Flags().String(optionNameFileName, "file", "file name template")
	cmd.Flags().Float64(optionNameFileSize, 1, "file size in MB")
	cmd.Flags().Int(optionNameNewNodeCount, 0, "number of new nodes")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	cmd.Flags().Int(optionNameStopNodeCount, 1, "number of nodes to stop")

	return cmd
}

package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/balances"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckBalances() *cobra.Command {
	const (
		optionNameUploadNodeCount    = "upload-node-count"
		optionNameFileName           = "file-name"
		optionNameFileSize           = "file-size"
		optionNameSeed               = "seed"
		optionNameDryRun             = "dry-run"
		optionNameWaitBeforeDownload = "wait-before-download"
	)

	var (
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "balances",
		Short: "Executes balances check",
		Long:  `Executes balances check.`,
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

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			if dryRun {
				return balances.DryRunCheck(cluster)
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			fileSize := round(c.config.GetFloat64(optionNameFileSize) * 1024 * 1024)

			return balances.Check(cluster, balances.Options{
				UploadNodeCount:    c.config.GetInt(optionNameUploadNodeCount),
				FileName:           c.config.GetString(optionNameFileName),
				FileSize:           fileSize,
				Seed:               seed,
				WaitBeforeDownload: c.config.GetInt(optionNameWaitBeforeDownload),
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload files to")
	cmd.Flags().String(optionNameFileName, "file", "file name template")
	cmd.Flags().Float64(optionNameFileSize, 1, "file size in MB")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	cmd.Flags().BoolVar(&dryRun, optionNameDryRun, false, "don't upload and download files, just validate")
	cmd.Flags().IntP(optionNameWaitBeforeDownload, "w", 5, "wait before downloading a file [s]")

	return cmd
}

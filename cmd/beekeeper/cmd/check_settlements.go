package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/settlements"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckSettlements() *cobra.Command {
	const (
		optionNameUploadNodeCount    = "upload-node-count"
		optionNameFileName           = "file-name"
		optionNameFileSize           = "file-size"
		optionNameSeed               = "seed"
		optionNameThreshold          = "threshold"
		optionNameDryRun             = "dry-run"
		optionNameWaitBeforeDownload = "wait-before-download"
		optionNameExpectSettlements  = "expect-settlements"
	)

	var (
		dryRun            bool
		expectSettlements bool
	)

	cmd := &cobra.Command{
		Use:   "settlements",
		Short: "Executes settlements check",
		Long:  `Executes settlements check.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			pusher := push.New(c.config.GetString(optionNamePushGateway), cfg.Cluster.Namespace)

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			fileSize := round(c.config.GetFloat64(optionNameFileSize) * 1024 * 1024)

			if dryRun {
				return settlements.DryRunCheck(cluster, settlements.Options{
					NodeGroup:          "bee",
					UploadNodeCount:    c.config.GetInt(optionNameUploadNodeCount),
					FileName:           c.config.GetString(optionNameFileName),
					FileSize:           fileSize,
					Seed:               seed,
					Threshold:          c.config.GetInt64(optionNameThreshold),
					WaitBeforeDownload: c.config.GetInt(optionNameWaitBeforeDownload),
					ExpectSettlements:  c.config.GetBool(optionNameExpectSettlements),
				})
			}

			return settlements.Check(cluster, settlements.Options{
				NodeGroup:          "bee",
				UploadNodeCount:    c.config.GetInt(optionNameUploadNodeCount),
				FileName:           c.config.GetString(optionNameFileName),
				FileSize:           fileSize,
				Seed:               seed,
				Threshold:          c.config.GetInt64(optionNameThreshold),
				WaitBeforeDownload: c.config.GetInt(optionNameWaitBeforeDownload),
				ExpectSettlements:  c.config.GetBool(optionNameExpectSettlements),
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload files to")
	cmd.Flags().String(optionNameFileName, "file", "file name template")
	cmd.Flags().Float64(optionNameFileSize, 2, "file size in MB")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	cmd.Flags().Int64P(optionNameThreshold, "t", 10000000000000, "balances treshold")
	cmd.Flags().BoolVar(&dryRun, optionNameDryRun, false, "don't upload and download files, just validate")
	cmd.Flags().IntP(optionNameWaitBeforeDownload, "w", 5, "wait before downloading a file [s]")
	cmd.Flags().BoolVar(&expectSettlements, optionNameExpectSettlements, true, "expects settlements happening during settlements check")

	return cmd
}

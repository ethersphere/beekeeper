package cmd

import (
	"errors"
	"fmt"

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
		optionNameKubeConfig        = "kubeconfig"
		optionNameHelmRelease       = "helm-release"
		optionNameHelmChart         = "helm-chart"
	)

	cmd := &cobra.Command{
		Use:   "fileretrievaldynamic",
		Short: "Checks file retrieval ability of the dynamic cluster",
		Long: `Checks file retrieval ability of the dynamic cluster.
It uploads given number of files to given number of nodes, 
and attempts retrieval of those files from the last node in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(c.config.GetString(optionNameKubeConfig)) == 0 {
				return errors.New("bad parameters: full path to KubeConfig must be provided")
			}
			if c.config.GetInt(optionNameDownloadNodeCount) <= 0 {
				return errors.New("bad parameters: download-node-count must be greater than 0")
			}
			l1 := 2*c.config.GetInt(optionNameDownloadNodeCount) + c.config.GetInt(optionNameStopNodeCount) + 2
			r1 := c.config.GetInt(optionNameNodeCount)
			if l1 > r1 {
				return fmt.Errorf("bad parameters: 2x download-node-count + stop-node-count + 2 must be <= node-count ; now: %d > %d", l1, r1)
			}
			l2 := 3*c.config.GetInt(optionNameDownloadNodeCount) + 2*c.config.GetInt(optionNameStopNodeCount) + 2
			r2 := c.config.GetInt(optionNameNodeCount) + c.config.GetInt(optionNameNewNodeCount)
			if l2 > r2 {
				return fmt.Errorf("bad parameters: 3x download-node-count + 2x stop-node-count + 2 must be <= node-count + new-node-count ; now: %d > %d", l2, r2)
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
				KubeConfig:        c.config.GetString(optionNameKubeConfig),
				HelmRelease:       c.config.GetString(optionNameHelmRelease),
				HelmChart:         c.config.GetString(optionNameHelmChart),
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
	cmd.Flags().String(optionNameKubeConfig, "", "full path to KubeConfig")
	cmd.Flags().String(optionNameHelmRelease, "bee", "helm release")
	cmd.Flags().String(optionNameHelmChart, "ethersphere/bee", "helm chart")

	return cmd
}

package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/chunkrepair"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initCheckChunkRepair() *cobra.Command {
	const (
		optionNameSeed       = "seed"
		optionNumberOfChunks = "number-of-chunks"
	)

	cmd := &cobra.Command{
		Use:   "chunkrepair",
		Short: "Checks chunk repair ability of the cluster",
		Long: `Checks chunk repair ability of the cluster.
It uploads given number of chunks to given number of nodes, 
and attempts repairing of those chunks for the other nodes in the cluster.`,
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

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			return chunkrepair.Check(cluster, chunkrepair.Options{
				NumberOfChunksToRepair: c.config.GetInt(optionNumberOfChunks),
				Seed:                   seed,
			}, pusher, c.config.GetBool(optionNamePushMetrics))

		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().Float64(optionNumberOfChunks, 1, "no of chunks to repair")
	return cmd
}

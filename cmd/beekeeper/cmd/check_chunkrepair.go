package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/chunkrepair"
	"github.com/ethersphere/beekeeper/pkg/config"
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

			return chunkrepair.Check(cluster, chunkrepair.Options{
				NodeGroup:              "bee",
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

package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/gc"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckGc() *cobra.Command {
	const (
		optionNameDbCapacity      = "db-capacity"
		optionNameDivisor         = "capacity-divisor"
		optionNameSeed            = "seed"
		optionNameWaitBeforeCheck = "wait"
	)

	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Checks that a node on the cluster flushes one chunk correctly.",
		Long:  "Checks that a node on the cluster flushes one chunk correctly.",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			return gc.CheckChunkNotFound(cluster, gc.Options{
				NodeGroup:        "bee",
				StoreSize:        c.config.GetInt(optionNameDbCapacity),
				StoreSizeDivisor: c.config.GetInt(optionNameDivisor),
				Wait:             c.config.GetInt(optionNameWaitBeforeCheck),
				Seed:             seed,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int(optionNameDbCapacity, 1000, "DB capacity in chunks")
	cmd.Flags().Int(optionNameDivisor, 3, "divide store size by which value when uploading bytes")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	cmd.Flags().IntP(optionNameWaitBeforeCheck, "w", 5, "wait before check")

	return cmd
}

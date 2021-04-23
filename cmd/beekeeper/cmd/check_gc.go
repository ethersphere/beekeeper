package cmd

import (
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/gc"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckGc() *cobra.Command {
	const (
		optionNameDbCapacity      = "db-capacity"
		optionNameDivisor         = "capacity-divisor"
		optionNameSeed            = "seed"
		optionNameWaitBeforeCheck = "wait"
		optionReserve             = "reserve"
		optionReserveSize         = "reserve-size"
	)

	var (
		runReserve bool
	)

	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Checks that a node on the cluster flushes one chunk correctly.",
		Long:  "Checks that a node on the cluster flushes one chunk correctly.",
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			cluster := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", *ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i), bee.NodeOptions{}); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			if runReserve {
				return gc.CheckReserve(cluster, gc.Options{
					NodeGroup:        "nodes",
					StoreSize:        c.config.GetInt(optionNameDbCapacity),
					StoreSizeDivisor: c.config.GetInt(optionNameDivisor),
					Wait:             c.config.GetDuration(optionNameWaitBeforeCheck),
					Seed:             seed,
					PostageAmount:    c.config.GetInt64(optionNamePostageAmount),
					PostageWait:      c.config.GetDuration(optionNamePostageBatchhWait),
					ReserveSize:      c.config.GetInt(optionReserveSize),
				})
			}

			return gc.CheckChunkNotFound(cluster, gc.Options{
				NodeGroup:        "nodes",
				StoreSize:        c.config.GetInt(optionNameDbCapacity),
				StoreSizeDivisor: c.config.GetInt(optionNameDivisor),
				Wait:             c.config.GetDuration(optionNameWaitBeforeCheck),
				Seed:             seed,
				PostageAmount:    c.config.GetInt64(optionNamePostageAmount),
				PostageWait:      c.config.GetDuration(optionNamePostageBatchhWait),
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int(optionNameDbCapacity, 1000, "DB capacity in chunks")
	cmd.Flags().Int(optionNameDivisor, 3, "divide store size by which value when uploading bytes")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	cmd.Flags().Duration(optionNameWaitBeforeCheck, time.Second*5, "wait before check")
	cmd.Flags().BoolVar(&runReserve, optionReserve, true, "run reserve check")
	cmd.Flags().Int(optionReserveSize, 1024, "reserve size of the node")

	return cmd
}

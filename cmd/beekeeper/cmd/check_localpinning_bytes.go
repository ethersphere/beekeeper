package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/localpinning"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckLocalPinningBytes() *cobra.Command {
	const (
		optionNameDbCapacity = "db-capacity"
		optionNameDivisor    = "capacity-divisor"
		optionNameSeed       = "seed"
	)

	cmdBytes := &cobra.Command{
		Use:   "pin-bytes",
		Short: "Checks that a node on the cluster pins bytes correctly.",
		Long:  "Checks that a node on the cluster pins bytes correctly.",
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

			return localpinning.CheckBytesFound(cluster, localpinning.Options{
				NodeGroup:        "nodes",
				StoreSize:        c.config.GetInt(optionNameDbCapacity),
				StoreSizeDivisor: c.config.GetInt(optionNameDivisor),
				Seed:             seed,
				PostageAmount:    c.config.GetInt64(optionNamePostageAmount),
				PostageWait:      c.config.GetDuration(optionNamePostageBatchhWait),
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmdBytes.Flags().Int(optionNameDbCapacity, 1000, "DB capacity in chunks")
	cmdBytes.Flags().Int(optionNameDivisor, 3, "divide store size by which value when uploading bytes")
	cmdBytes.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	return cmdBytes
}

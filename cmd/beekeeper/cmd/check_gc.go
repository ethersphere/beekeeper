package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/gc"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckGc() *cobra.Command {
	const (
		optionNameDbCapacity = "db-capacity"
		optionNameDivisor    = "capacity-divisor"
		optionNameSeed       = "seed"
	)

	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Checks that a node on the cluster flushes one chunk correctly.",
		Long:  "Checks that a node on the cluster flushes one chunk correctly.",
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			cluster, err := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				InCluster:           c.config.GetBool(optionNameInCluster),
				KubeconfigPath:      c.config.GetString(optionNameKubeconfig),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})
			if err != nil {
				return fmt.Errorf("creating new Bee cluster: %v", err)
			}

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", *ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i)); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			return gc.CheckChunkNotFound(cluster, gc.Options{
				NodeGroup:        "nodes",
				StoreSize:        c.config.GetInt(optionNameDbCapacity),
				StoreSizeDivisor: c.config.GetInt(optionNameDivisor),
				Seed:             seed,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int(optionNameDbCapacity, 1000, "DB capacity in chunks")
	cmd.Flags().Int(optionNameDivisor, 3, "divide store size by which value when uploading bytes")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")

	return cmd
}

package cmd

import (
	"fmt"

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
			cluster := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				KubeconfigPath:      c.config.GetString(optionNameStartKubeconfig),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", *ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i)); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			return chunkrepair.Check(cluster, chunkrepair.Options{
				NodeGroup:              "nodes",
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

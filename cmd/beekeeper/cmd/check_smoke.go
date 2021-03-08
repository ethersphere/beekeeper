package cmd

import (
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/smoke"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckSmoke() *cobra.Command {
	const (
		optionNameRuns      = "runs"
		optionNameBytes     = "bytes"
		optionNameMegabytes = "megabytes"
		optionNameSeed      = "seed"
	)

	var (
		runs int
	)

	cmd := &cobra.Command{
		Use:   "smoke",
		Short: "Runs a simple smoke test over the cluster",
		Long: `Runs a smoke test that picks a random node from the cluster,
		uploads random data with the predefined size to it then tries to download
		it from another random node.`,
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

			var b, mb = c.config.GetInt(optionNameBytes), c.config.GetInt(optionNameMegabytes)
			if b == 0 && mb == 0 {
				return errors.New("must set either bytes or megabytes")
			}

			if b == 0 && mb > 0 {
				b = mb * 1000 * 1000
			}

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

			return smoke.Check(cluster, smoke.Options{
				NodeGroup:       "nodes",
				UploadNodeCount: c.config.GetInt(optionNameNodeCount),
				Seed:            seed,
				Runs:            runs,
				Bytes:           b,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntVarP(&runs, optionNameRuns, "r", 1, "number of runs to do")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().IntP(optionNameBytes, "b", 0, "number of bytes to upload on each run")
	cmd.Flags().IntP(optionNameMegabytes, "m", 0, "number of megabytes to upload on each run")

	return cmd
}

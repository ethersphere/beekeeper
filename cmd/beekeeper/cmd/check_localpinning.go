package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/localpinning"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckLocalPinning() *cobra.Command {
	const (
		optionNameDBCapacity = "db-capacity"
		optionNameFileName   = "file-name"
		optionNameDivisor    = "capacity-divisor"
		optionNameSeed       = "seed"
	)

	cmd := &cobra.Command{
		Use:   "localpinning-1",
		Short: "Checks that a node on the cluster flushes one chunk correctly.",
		Long:  "Checks that a node on the cluster flushes one chunk correctly.",
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

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			return localpinning.CheckChunkNotFound(cluster, localpinning.Options{
				Seed: seed,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Float64(optionNameDBCapacity, 500, "DB capacity in chunks")
	cmd.Flags().Int(optionNameDivisor, 3, "divide store size by which value when uploading bytes")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")

	return cmd
}

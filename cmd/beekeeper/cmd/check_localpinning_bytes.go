package cmd

import (
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

			return localpinning.CheckBytesFound(cluster, localpinning.Options{
				StoreSize:        c.config.GetInt(optionNameDbCapacity),
				StoreSizeDivisor: c.config.GetInt(optionNameDivisor),
				Seed:             seed,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmdBytes.Flags().Int(optionNameDbCapacity, 1000, "DB capacity in chunks")
	cmdBytes.Flags().Int(optionNameDivisor, 3, "divide store size by which value when uploading bytes")
	cmdBytes.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	return cmdBytes
}

package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/spf13/cobra"
)

func (c *command) initCheckPing() *cobra.Command {
	const (
		optionNameSeed = "seed"
	)

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			runOptions := check.RunOptions{
				Seed: seed,
			}
			return check.Run(cmd.Context(), runOptions)
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")

	return cmd
}

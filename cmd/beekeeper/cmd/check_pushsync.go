package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPushSync() *cobra.Command {
	const (
		optionNameSeed       = "seed"
		optionNameRandomSeed = "random-seed"
	)

	cmd := &cobra.Command{
		Use:   "pushsync",
		Short: "Checks push sync",
		Long:  `Checks push sync`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PushSync(check.PushSyncOptions{
				DebugAPIURLTemplate: c.config.GetString(optionNameDebugAPIURLTemplate),
				Namespace:           c.config.GetString(optionNameNamespace),
				NodeCount:           c.config.GetInt(optionNameNodeCount),
				Seed:                c.config.GetInt64(optionNameSeed),
				RandomSeed:          c.config.GetBool(optionNameRandomSeed),
			})
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 1, "seed")
	cmd.Flags().BoolP(optionNameRandomSeed, "r", true, "random seed")

	return cmd
}

package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPushSync() *cobra.Command {
	const (
		optionNameNodeCount   = "node-count"
		optionNameNamespace   = "namespace"
		optionNameSeed        = "seed"
		optionNameRandomSeed  = "random-seed"
		optionNameURLTemplate = "url-template"
	)

	cmd := &cobra.Command{
		Use:   "pushsync",
		Short: "Checks push sync",
		Long:  `Checks push sync`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PushSync(check.PushSyncOptions{
				NodeCount:   c.config.GetInt(optionNameNodeCount),
				Namespace:   c.config.GetString(optionNameNamespace),
				Seed:        c.config.GetInt64(optionNameSeed),
				RandomSeed:  c.config.GetBool(optionNameRandomSeed),
				URLTemplate: c.config.GetString(optionNameURLTemplate),
			})
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flag(optionNameURLTemplate).Value.String() == "" {
				if err := cmd.MarkFlagRequired(optionNameNamespace); err != nil {
					panic(err)
				}
			}
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().IntP(optionNameNodeCount, "c", 1, "node count")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace")
	cmd.Flags().Int64P(optionNameSeed, "s", 1, "seed")
	cmd.Flags().BoolP(optionNameRandomSeed, "r", true, "random seed")
	cmd.Flags().StringP(optionNameURLTemplate, "u", "", "URL template")
	if err := cmd.Flags().MarkHidden(optionNameURLTemplate); err != nil {
		panic(err)
	}

	return cmd
}

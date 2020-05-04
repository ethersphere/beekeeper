package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPingPong() *cobra.Command {
	const (
		optionNameNodeCount   = "node-count"
		optionNameNamespace   = "namespace"
		optionNameSeed        = "seed"
		optionNameRandomSeed  = "random-seed"
		optionNameURLTemplate = "url-template"
	)

	cmd := &cobra.Command{
		Use:   "pingpong",
		Short: "Checks pingpong",
		Long:  `Checks pingpong`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PingPong(check.PingPongOptions{
				NodeCount:   c.config.GetInt(optionNameNodeCount),
				Namespace:   c.config.GetString(optionNameNamespace),
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
	cmd.Flags().StringP(optionNameURLTemplate, "u", "", "URL template")
	if err := cmd.Flags().MarkHidden(optionNameURLTemplate); err != nil {
		panic(err)
	}

	return cmd
}

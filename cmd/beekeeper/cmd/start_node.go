package cmd

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initStartNode() *cobra.Command {
	const (
		optionNameStartStandalone = "standalone"
	)

	var (
		standalone bool
	)

	cmd := &cobra.Command{
		Use:   "node",
		Short: "Start Bee node",
		Long:  `Start Bee node.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx := context.Background()

			node := bee.NewClient(bee.ClientOptions{KubeconfigPath: c.config.GetString(optionNameStartConfig)})

			return node.Start(ctx, standalone)
		},
		PreRunE: c.startPreRunE,
	}

	cmd.PersistentFlags().BoolVarP(&standalone, optionNameStartStandalone, "s", false, "start a standalone node")

	return cmd
}

package cmd

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initCreateNode() *cobra.Command {
	return &cobra.Command{
		Use:   "node",
		Short: "Create Bee node",
		Long:  `Create Bee node.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx := context.Background()
			standalone := true
			node := bee.NewClient(bee.ClientOptions{KubeconfigPath: c.config.GetString(optionNameK8SConfig)})
			node.Create(ctx, standalone)

			return
		},
		PreRunE: c.createPreRunE,
	}
}

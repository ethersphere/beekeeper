package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/spf13/cobra"
)

func (c *command) initCheckNodes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "Check nodes",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			opts := check.Options{
				NodeCount:       c.config.GetInt(optionNameNodeCount),
				NodeURLTemplate: c.config.GetString(optionNameNodeURLTemplate),
			}

			if err = check.Nodes(opts); err != nil {
				return err
			}

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	return cmd
}

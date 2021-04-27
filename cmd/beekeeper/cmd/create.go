package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initCreateCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create Bee infrastructure",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.AddCommand(c.initCreateK8SNamespace())
	cmd.AddCommand(c.initCreateBeeCluster())

	c.root.AddCommand(cmd)

	return nil
}

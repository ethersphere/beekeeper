package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initCreateCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "creates Bee infrastructure",
		Long:  `Creates Bee infrastructure.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
	}

	cmd.AddCommand(c.initCreateK8SNamespace())
	cmd.AddCommand(c.initCreateBeeCluster())

	c.root.AddCommand(cmd)

	return nil
}

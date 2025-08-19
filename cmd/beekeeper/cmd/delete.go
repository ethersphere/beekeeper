package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initDeleteCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes Bee infrastructure",
		Long:  `Deletes Bee infrastructure.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
	}

	cmd.AddCommand(c.initDeleteK8SNamespace())
	cmd.AddCommand(c.initDeleteBeeCluster())

	c.root.AddCommand(cmd)

	return nil
}

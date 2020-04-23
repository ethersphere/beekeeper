package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initCheckCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check Bees",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
	}

	cmd.AddCommand(c.initCheckPeerCount())

	c.root.AddCommand(cmd)
	return nil
}

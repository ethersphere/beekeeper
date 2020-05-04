package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initCheckCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run tests on Bee node(s)",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
	}

	cmd.AddCommand(c.initCheckPeerCount())
	cmd.AddCommand(c.initCheckPingPong())
	cmd.AddCommand(c.initCheckPushSync())

	c.root.AddCommand(cmd)
	return nil
}

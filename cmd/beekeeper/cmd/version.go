package cmd

import (
	"github.com/ethersphere/beekeeper"

	"github.com/spf13/cobra"
)

func (c *command) initVersionCmd() {
	c.root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Prints version number",
		Long: `Prints the current version number of the Beekeeper tool.

This command displays the semantic version of Beekeeper you are currently running.
Useful for verifying which version is installed, troubleshooting compatibility issues,
or ensuring you're running the expected version for your environment.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(beekeeper.Version)
		},
	})
}

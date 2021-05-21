package cmd

import (
	"github.com/ethersphere/beekeeper"

	"github.com/spf13/cobra"
)

func (c *command) initVersionCmd() {
	c.root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "prints version number",
		Long:  `Prints version number.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(beekeeper.Version)
		},
	})
}

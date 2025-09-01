package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initStakeCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "stake",
		Short: "Stakes Bee nodes",
		Long:  `Stakes Bee nodes with BZZ tokens and ETH for Bee node operations.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return nil
		},
		PreRunE: c.preRunE,
	}

	c.root.AddCommand(cmd)

	return nil
}

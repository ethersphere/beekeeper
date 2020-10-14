package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *command) initCreateCluster() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster",
		Short: "Create Bee cluster",
		Long:  `Create Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fmt.Println("create cluster")
			return
		},
		PreRunE: c.createPreRunE,
	}
}

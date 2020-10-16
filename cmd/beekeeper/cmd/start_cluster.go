package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *command) initStartCluster() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster",
		Short: "Start Bee cluster",
		Long:  `Start Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fmt.Println("start cluster")
			return
		},
		PreRunE: c.startPreRunE,
	}
}

package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/spf13/cobra"
)

func (c *command) initK8SCheck() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "k8s check",
		Long:  `k8s check.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fmt.Println("k8s")

			return k8s.Check()
		},
		PreRunE: c.k8sPreRunE,
	}
}

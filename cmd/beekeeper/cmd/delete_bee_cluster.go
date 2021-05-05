package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initDeleteBeeCluster() *cobra.Command {
	const (
		optionNameClusterName = "cluster-name"
		// optionNameTimeout        = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "bee-cluster",
		Short: "Delete Bee cluster",
		Long:  `Delete Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.deleteCluster(cmd.Context(), c.globalConfig.GetString(optionNameClusterName), c.config)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.globalConfig.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")

	c.root.AddCommand(cmd)

	return cmd
}

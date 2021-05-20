package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initDeleteBeeCluster() *cobra.Command {
	const (
		optionNameClusterName = "cluster-name"
		optionNameWithStorage = "with-storage"
		// TODO: optionNameTimeout        = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "bee-cluster",
		Short: "Delete Bee cluster",
		Long:  `Delete Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.deleteCluster(cmd.Context(), c.globalConfig.GetString(optionNameClusterName), c.config, c.globalConfig.GetBool(optionNameWithStorage))
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")
	cmd.Flags().Bool(optionNameWithStorage, false, "delete storage")

	return cmd
}

package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

func (c *command) initDeleteBeeCluster() *cobra.Command {
	const (
		optionNameClusterName = "cluster-name"
		optionNameWithStorage = "with-storage"
		optionNameTimeout     = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "bee-cluster",
		Short: "Delete Bee cluster",
		Long:  `Delete Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			return c.deleteCluster(ctx, c.globalConfig.GetString(optionNameClusterName), c.config, c.globalConfig.GetBool(optionNameWithStorage))
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")
	cmd.Flags().Bool(optionNameWithStorage, false, "delete storage")
	cmd.Flags().Duration(optionNameTimeout, 15*time.Minute, "timeout")

	return cmd
}

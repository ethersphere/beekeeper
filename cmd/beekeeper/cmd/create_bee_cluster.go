package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initCreateBeeCluster() *cobra.Command {
	const (
		optionNameClusterName = "cluster-name"
		// TODO: optionNameTimeout        = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "bee-cluster",
		Short: "Create Bee cluster",
		Long:  `Create Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			_, err = c.setupCluster(cmd.Context(), c.globalConfig.GetString(optionNameClusterName), c.config, true)

			return err
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")

	return cmd
}

package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/config"
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
			cfg, err := config.Read("config/config.yaml")
			if err != nil {
				return err
			}

			return c.deleteCluster(cmd.Context(), c.config.GetString(optionNameClusterName), cfg)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")

	c.root.AddCommand(cmd)

	return cmd
}

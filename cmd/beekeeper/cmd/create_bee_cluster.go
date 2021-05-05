package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initCreateBeeCluster() *cobra.Command {
	const (
		optionNameClusterName = "cluster-name"
		// optionNameTimeout        = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "bee-cluster",
		Short: "Create Bee cluster",
		Long:  `Create Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config/config.yaml")
			if err != nil {
				return err
			}

			_, err = c.setupCluster(cmd.Context(), c.config.GetString(optionNameClusterName), cfg, true)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			return
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")

	c.root.AddCommand(cmd)

	return cmd
}

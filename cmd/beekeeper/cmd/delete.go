package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initDeleteCmd() (err error) {
	const (
		optionNameClusterName = "cluster-name"
		// optionNameTimeout        = "timeout"
	)

	var (
		clusterName string
		// timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete Bee",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}

			return deleteCluster(cmd.Context(), clusterName, cfg)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "default", "cluster name")

	c.root.AddCommand(cmd)

	return nil
}

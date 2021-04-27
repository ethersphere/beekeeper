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

	var (
		clusterName string
		// timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "bee-cluster",
		Short: "Delete Bee cluster",
		Long:  `Delete Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}

			return deleteCluster(cmd.Context(), clusterName, cfg)
		},
	}

	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "default", "cluster name")

	return cmd
}

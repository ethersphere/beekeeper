package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initStartCmd() (err error) {
	const (
		optionNameClusterName = "cluster-name"
		// optionNameTimeout        = "timeout"
	)

	var (
		clusterName string
		// timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Bee",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}

			_, err = setupCluster(cmd.Context(), clusterName, cfg, true)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			return
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "default", "cluster name")

	c.root.AddCommand(cmd)

	return nil
}

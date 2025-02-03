package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

const (
	optionNameClusterName string = "cluster-name"
	optionNameWalletKey   string = "wallet-key"
	optionNameTimeout     string = "timeout"
)

func (c *command) initCreateBeeCluster() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bee-cluster",
		Short: "creates Bee cluster",
		Long:  `creates Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()
			start := time.Now()
			_, err = c.setupCluster(ctx, c.globalConfig.GetString(optionNameClusterName), true)
			c.log.Infof("cluster setup took %s", time.Since(start))
			return err
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "cluster name")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")

	return cmd
}

package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

const (
	optionNameClusterName       = "cluster-name"
	optionNameChainNodeEndpoint = "geth-url"
	optionNameWalletKey         = "wallet-key"
	optionNameTimeout           = "timeout"
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
			_, err = c.setupCluster(ctx, c.globalConfig.GetString(optionNameClusterName), c.config, true)
			c.log.Infof("cluster setup took %s", time.Since(start))
			return err
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")
	cmd.Flags().String(optionNameChainNodeEndpoint, "", "Endpoint to chain node. Required.")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")

	return cmd
}

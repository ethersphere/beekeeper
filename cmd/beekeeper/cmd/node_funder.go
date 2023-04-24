package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initNodeFunderCmd() (err error) {
	const (
		optionNameAddresses         = "addresses"
		optionNameNamespace         = "namespace"
		optionNameChainNodeEndpoint = "chain-node-endpoint"
		optionNameWalletKey         = "wallet-key"
		optionNameMinNative         = "min-native"
		optionNameMinSwarm          = "min-swarm"
		optionNameTimeout           = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "node-funder",
		Short: "funds bee nodes with ETH and BZZ",
		Long:  `Fund makes BZZ tokens and ETH deposits to given Ethereum addresses. beekeeper node-funder`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.NodeFunder{}

			// namespace check
			if cfg.Namespace = c.globalConfig.GetString(optionNameNamespace); cfg.Namespace == "" {
				return fmt.Errorf("namespace not provided")
			}

			// addresses check
			if cfg.Addresses = c.globalConfig.GetStringSlice(optionNameAddresses); len(cfg.Addresses) < 1 {
				return fmt.Errorf("bee node addresses not provided")
			}

			// chain node endpoint check
			if cfg.ChainNodeEndpoint = c.globalConfig.GetString(optionNameChainNodeEndpoint); cfg.ChainNodeEndpoint == "" {
				return fmt.Errorf("chain node endpoint not provided")
			}

			// wallet key check
			if cfg.WalletKey = c.globalConfig.GetString(optionNameWalletKey); cfg.WalletKey == "" {
				return fmt.Errorf("wallet key not provided")
			}

			cfg.MinAmounts.NativeCoin = c.globalConfig.GetFloat64(optionNameMinNative)
			cfg.MinAmounts.SwarmToken = c.globalConfig.GetFloat64(optionNameMinSwarm)

			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			return c.fund(ctx, cfg)
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameNamespace, "", "kubernetes namespace")
	cmd.Flags().String(optionNameChainNodeEndpoint, "", "endpoint to chain node")
	cmd.Flags().String(optionNameWalletKey, "", "wallet key")
	cmd.Flags().Float64(optionNameMinNative, 0, "specifies min amout of chain native coins (ETH) nodes should have")
	cmd.Flags().Float64(optionNameMinSwarm, 0, "specifies min amout of swarm tokens (BZZ) nodes should have")
	cmd.Flags().StringSlice(optionNameAddresses, nil, "Bee node addresses (must start with 0x)")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) fund(ctx context.Context, cfg config.NodeFunder) (err error) {
	c.logger.Debugf("funding nodes with config: %+v", cfg)

	if cfg.Namespace != "" {
		return c.fundNamespace(ctx, cfg)
	}

	return c.fundAddresses(ctx, cfg)
}

func (c *command) fundNamespace(ctx context.Context, cfg config.NodeFunder) (err error) {
	c.logger.Infof("funding nodes in namespace %s", cfg.Namespace)

	return nil
}

func (c *command) fundAddresses(ctx context.Context, cfg config.NodeFunder) (err error) {
	c.logger.Infof("funding addresses in namespace %s", cfg.Namespace)

	return nil
}

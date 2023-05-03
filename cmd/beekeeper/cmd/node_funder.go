package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/node-funder/pkg/funder"
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

			cfg.Namespace = c.globalConfig.GetString(optionNameNamespace)
			cfg.Addresses = c.globalConfig.GetStringSlice(optionNameAddresses)

			// namespace and addresses check
			if cfg.Namespace == "" && len(cfg.Addresses) == 0 {
				return fmt.Errorf("namespace or addresses not provided")
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

			// TODO: add timeout to node-funder
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			return funder.Fund(ctx, funder.Config{
				Namespace:         cfg.Namespace,
				Addresses:         cfg.Addresses,
				ChainNodeEndpoint: cfg.ChainNodeEndpoint,
				WalletKey:         cfg.WalletKey,
				MinAmounts: funder.MinAmounts{
					NativeCoin: cfg.MinAmounts.NativeCoin,
					SwarmToken: cfg.MinAmounts.SwarmToken,
				},
			})
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

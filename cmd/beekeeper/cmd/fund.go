package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/ethersphere/beekeeper/pkg/swap"
	"github.com/spf13/cobra"
)

func (c *command) initFundCmd() (err error) {
	const (
		optionNameAddresses  = "addresses"
		optionNameBzzDeposit = "bzz-deposit"
		optionNameEthDeposit = "eth-deposit"
		optionNameGethURL    = "geth-url"
		optionNameTimeout    = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "fund",
		Short: "Fund Ethereum addresses",
		Long: `Fund makes BZZ tokens and ETH deposits to given Ethereum addresses.
beekeeper fund --addresses=0xf176839c150e52fe30e5c2b5c648465c6fdfa532,0xebe269e07161c68a942a3a7fce6b4ed66867d6f0
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(c.globalConfig.GetStringSlice(optionNameAddresses)) < 1 {
				return fmt.Errorf("bee node Ethereum addresses not provided")
			}

			gethUrl, err := url.Parse(c.globalConfig.GetString(optionNameGethURL))
			if err != nil {
				return fmt.Errorf("parsing Geth URL: %w", err)
			}

			geth := swap.NewGethClient(gethUrl, &swap.GethClientOptions{})

			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			for _, a := range c.globalConfig.GetStringSlice(optionNameAddresses) {
				if err := geth.Fund(ctx, a, c.globalConfig.GetInt64(optionNameEthDeposit), c.globalConfig.GetInt64(optionNameBzzDeposit)); err != nil {
					return fmt.Errorf("funding Ethereum address %s: %w", a, err)
				}
			}

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.globalConfig.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().StringSlice(optionNameAddresses, nil, "Bee node Ethereum addresses (must start with 0x)")
	cmd.Flags().Int64(optionNameBzzDeposit, 1000000000000000000, "BZZ tokens amount to deposit")
	cmd.Flags().Int64(optionNameEthDeposit, 1000000000000000000, "ETH amount to deposit")
	cmd.Flags().String(optionNameGethURL, "http://geth.beekeeper.staging.internal", "Geth node URL")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func (c *command) initFundCmd() (err error) {
	const (
		optionNameAddresses       = "addresses"
		optionNameBzzDeposit      = "bzz-deposit"
		optionNameBzzTokenAddress = "bzz-token-address"
		optionNameEthAccount      = "eth-account"
		optionNameEthDeposit      = "eth-deposit"
		optionNameGethURL         = "geth-url"
		optionNameTimeout         = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "fund",
		Short: "funds Ethereum addresses",
		Long: `Fund makes BZZ tokens and ETH deposits to given Ethereum addresses.
beekeeper fund --addresses=0xf176839c150e52fe30e5c2b5c648465c6fdfa532,0xebe269e07161c68a942a3a7fce6b4ed66867d6f0`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(c.globalConfig.GetStringSlice(optionNameAddresses)) < 1 {
				return fmt.Errorf("bee node Ethereum addresses not provided")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			for _, a := range c.globalConfig.GetStringSlice(optionNameAddresses) {
				ethDeposit := c.globalConfig.GetFloat64(optionNameEthDeposit)
				tx, err := c.swapClient.SendETH(ctx, a, ethDeposit)
				if err != nil {
					return fmt.Errorf("send eth: %w", err)
				}
				fmt.Printf("%s funded with %.2f ETH, transaction: %s\n", a, ethDeposit, tx)

				bzzDeposit := c.globalConfig.GetFloat64(optionNameBzzDeposit)
				tx, err = c.swapClient.SendBZZ(ctx, a, bzzDeposit)
				if err != nil {
					return fmt.Errorf("deposit bzz: %w", err)
				}
				fmt.Printf("%s funded with %.2f BZZ, transaction: %s\n", a, bzzDeposit, tx)
			}

			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringSlice(optionNameAddresses, nil, "Bee node Ethereum addresses (must start with 0x)")
	cmd.Flags().Float64(optionNameBzzDeposit, 100.0, "BZZ tokens amount to deposit")
	cmd.Flags().String(optionNameBzzTokenAddress, "0x6aab14fe9cccd64a502d23842d916eb5321c26e7", "BZZ token address")
	cmd.Flags().String(optionNameEthAccount, "0x62cab2b3b55f341f10348720ca18063cdb779ad5", "ETH account address")
	cmd.Flags().Float64(optionNameEthDeposit, 1.0, "ETH amount to deposit")
	cmd.Flags().String(optionNameGethURL, "http://geth-swap.geth-swap.dai.internal", "Geth node URL")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

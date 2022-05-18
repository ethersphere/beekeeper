package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/orchestration/utils"
	"github.com/spf13/cobra"
)

func (c *command) initFundCmd() (err error) {
	const (
		optionNameAddresses       = "addresses"
		optionNameAddressCreate   = "address-create"
		optionNameAddressCount    = "address-count"
		optionNameEthAccount      = "eth-account"
		optionNameBzzTokenAddress = "bzz-token-address"
		optionNameGethURL         = "geth-url"
		optionNameBzzDeposit      = "bzz-deposit"
		optionNameEthDeposit      = "eth-deposit"
		optionNameGBzzDeposit     = "gBzz-deposit"
		optionNamePassword        = "password"
		optionNameTimeout         = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "fund",
		Short: "funds Ethereum addresses",
		Long: `Fund makes BZZ tokens and ETH deposits to given Ethereum addresses.
beekeeper fund --addresses=0xf176839c150e52fe30e5c2b5c648465c6fdfa532,0xebe269e07161c68a942a3a7fce6b4ed66867d6f0 --bzz-deposit 100.0 --eth-deposit 0.1
beekeeper fund --address-create --address-count 2 --bzz-deposit 100.0 --eth-deposit 0.1`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var addresses []string
			if c.globalConfig.GetBool(optionNameAddressCreate) {
				for i := 0; i < c.globalConfig.GetInt(optionNameAddressCount); i++ {
					swarmKey, err := utils.CreateSwarmKey(c.globalConfig.GetString(optionNamePassword))
					if err != nil {
						return fmt.Errorf("creating Swarm key: %w", err)
					}

					var key struct{ Address string }
					if err := json.Unmarshal([]byte(swarmKey), &key); err != nil {
						return fmt.Errorf("unmarshaling Swarm address: %w", err)
					}
					addresses = append(addresses, "0x"+key.Address)
				}
			} else if len(c.globalConfig.GetStringSlice(optionNameAddresses)) < 1 {
				return fmt.Errorf("bee node Ethereum addresses not provided")
			} else {
				addresses = c.globalConfig.GetStringSlice(optionNameAddresses)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			for _, a := range addresses {
				fmt.Printf("address: %s\n", a)
				// ETH funding
				ethDeposit := c.globalConfig.GetFloat64(optionNameEthDeposit)
				if ethDeposit > 0 {
					tx, err := c.swapClient.SendETH(ctx, a, ethDeposit)
					if err != nil {
						return fmt.Errorf("send eth: %w", err)
					}
					fmt.Printf("%s funded with %.2f ETH, transaction: %s\n", a, ethDeposit, tx)
				}
				// BZZ funding
				bzzDeposit := c.globalConfig.GetFloat64(optionNameBzzDeposit)
				if bzzDeposit > 0 {
					tx, err := c.swapClient.SendBZZ(ctx, a, bzzDeposit)
					if err != nil {
						return fmt.Errorf("deposit bzz: %w", err)
					}
					fmt.Printf("%s funded with %.2f BZZ, transaction: %s\n", a, bzzDeposit, tx)
				}
				// gBZZ funding
				gBzzDeposit := c.globalConfig.GetFloat64(optionNameGBzzDeposit)
				if gBzzDeposit > 0 {
					tx, err := c.swapClient.SendGBZZ(ctx, a, gBzzDeposit)
					if err != nil {
						return fmt.Errorf("deposit gBzz: %w", err)
					}
					fmt.Printf("%s funded with %.2f gBZZ, transaction: %s\n", a, gBzzDeposit, tx)
				}
			}

			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringSlice(optionNameAddresses, nil, "Bee node Ethereum addresses (must start with 0x)")
	cmd.Flags().Bool(optionNameAddressCreate, false, "if enabled, creates Ethereum address(es)")
	cmd.Flags().Int(optionNameAddressCount, 1, "number of Ethereum addresses to create")
	cmd.Flags().String(optionNameBzzTokenAddress, "0x6aab14fe9cccd64a502d23842d916eb5321c26e7", "BZZ token address")
	cmd.Flags().String(optionNameEthAccount, "0x62cab2b3b55f341f10348720ca18063cdb779ad5", "ETH account address")
	cmd.Flags().String(optionNameGethURL, "http://geth-swap.geth-swap.staging.internal", "Geth node URL")
	cmd.Flags().Float64(optionNameBzzDeposit, 0, "BZZ tokens amount to deposit")
	cmd.Flags().Float64(optionNameGBzzDeposit, 0, "gBZZ tokens amount to deposit")
	cmd.Flags().Float64(optionNameEthDeposit, 0, "ETH amount to deposit")
	cmd.Flags().String(optionNamePassword, "beekeeper", "password for generating Ethereum addresses")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

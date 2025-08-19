package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

func (c *command) initFundCmd() (err error) {
	const (
		optionNameAddresses       = "addresses"
		optionNameAddressCreate   = "address-create"
		optionNameAddressCount    = "address-count"
		optionNameEthAccount      = "eth-account"
		optionNameBzzTokenAddress = "bzz-token-address"
		optionNameBzzDeposit      = "bzz-deposit"
		optionNameEthDeposit      = "eth-deposit"
		optionNameGBzzDeposit     = "gBzz-deposit"
		optionNamePassword        = "password"
		optionNamePrintKeys       = "print-keys"
		optionNamePrintAddresses  = "print-addresses"
		optionNameTimeout         = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "fund",
		Short: "Funds Ethereum addresses",
		Long: `Fund makes BZZ tokens and ETH deposits to given Ethereum addresses.
beekeeper fund --addresses=0xf176839c150e52fe30e5c2b5c648465c6fdfa532,0xebe269e07161c68a942a3a7fce6b4ed66867d6f0 --bzz-deposit 100.0 --eth-deposit 0.1
beekeeper fund --address-create --address-count 2 --bzz-deposit 100.0 --eth-deposit 0.1`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var addresses []string
			var createdKeys []orchestration.EncryptedKey
			if c.globalConfig.GetBool(optionNameAddressCreate) {
				for i := 0; i < c.globalConfig.GetInt(optionNameAddressCount); i++ {
					key, err := orchestration.NewEncryptedKey(c.globalConfig.GetString(optionNamePassword))
					if err != nil {
						return fmt.Errorf("creating Swarm key: %w", err)
					}

					addresses = append(addresses, "0x"+key.Address)
					createdKeys = append(createdKeys, *key)
				}
				if c.globalConfig.GetBool(optionNamePrintKeys) {
					k, err := json.Marshal(createdKeys)
					if err != nil {
						return fmt.Errorf("marshaling Swarm keys: %w", err)
					}
					c.log.Infof("%s", k)
				}
			} else if len(c.globalConfig.GetStringSlice(optionNameAddresses)) < 1 {
				return fmt.Errorf("bee node Ethereum addresses not provided")
			} else {
				addresses = c.globalConfig.GetStringSlice(optionNameAddresses)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			for _, a := range addresses {
				if c.globalConfig.GetBool(optionNamePrintAddresses) {
					c.log.Infof("address: %s", a)
				}
				// ETH funding
				ethDeposit := c.globalConfig.GetFloat64(optionNameEthDeposit)
				if ethDeposit > 0 {
					tx, err := c.swapClient.SendETH(ctx, a, ethDeposit)
					if err != nil {
						return fmt.Errorf("send eth: %w", err)
					}
					c.log.Infof("%s funded with %.2f ETH, transaction: %s\n", a, ethDeposit, tx)
				}
				// BZZ funding
				bzzDeposit := c.globalConfig.GetFloat64(optionNameBzzDeposit)
				if bzzDeposit > 0 {
					tx, err := c.swapClient.SendBZZ(ctx, a, bzzDeposit)
					if err != nil {
						return fmt.Errorf("deposit bzz: %w", err)
					}
					c.log.Infof("%s funded with %.2f BZZ, transaction: %s\n", a, bzzDeposit, tx)
				}
				// gBZZ funding
				gBzzDeposit := c.globalConfig.GetFloat64(optionNameGBzzDeposit)
				if gBzzDeposit > 0 {
					tx, err := c.swapClient.SendGBZZ(ctx, a, gBzzDeposit)
					if err != nil {
						return fmt.Errorf("deposit gBzz: %w", err)
					}
					c.log.Infof("%s funded with %.2f gBZZ, transaction: %s\n", a, gBzzDeposit, tx)
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
	cmd.Flags().Float64(optionNameBzzDeposit, 0, "BZZ tokens amount to deposit")
	cmd.Flags().Float64(optionNameGBzzDeposit, 0, "gBZZ tokens amount to deposit")
	cmd.Flags().Float64(optionNameEthDeposit, 0, "ETH amount to deposit")
	cmd.Flags().String(optionNamePassword, "beekeeper", "password for generating Ethereum addresses")
	cmd.Flags().Bool(optionNamePrintKeys, false, "if enabled, prints created keys")
	cmd.Flags().Bool(optionNamePrintAddresses, false, "if enabled, prints funded addresses")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

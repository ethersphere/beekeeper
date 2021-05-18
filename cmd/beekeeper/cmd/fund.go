package cmd

import (
	"fmt"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/swap"
	"github.com/spf13/cobra"
)

// curl -H 'Content-Type: application/json' --data '{"jsonrpc":"2.0", "method": "personal_unlockAccount", "params": ["0x62cab2b3b55f341f10348720ca18063cdb779ad5", "", 0], "id":"1"}' http://geth.beekeeper.staging.internal/

func (c *command) initFundCmd() (err error) {
	const (
		optionNameClusterName = "cluster-name"
		// TODO: optionNameTimeout        = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "fund",
		Short: "Fund Bee node",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fmt.Println("fund")

			baseUrl, err := url.Parse("http://geth.beekeeper.staging.internal")
			if err != nil {
				return err
			}

			geth := swap.NewGethClient(baseUrl, &swap.GethClientOptions{})

			ethDeposit := 1
			tokenDeposit := 1000
			geth.Fund(cmd.Context(), "0xf176839c150e52fe30e5c2b5c648465c6fdfa532", ethDeposit, tokenDeposit)
			geth.Fund(cmd.Context(), "0xebe269e07161c68a942a3a7fce6b4ed66867d6f0", ethDeposit, tokenDeposit)

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.globalConfig.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().String(optionNameClusterName, "default", "cluster name")

	c.root.AddCommand(cmd)

	return nil
}

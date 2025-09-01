package cmd

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/spf13/cobra"
)

const (
	optionNameAmount = "amount"
)

var (
	errMissingStakeAmount = errors.New("stake amount not provided")
	errInvalidAmount      = errors.New("invalid stake amount")
)

func (c *command) initStakeCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "stake",
		Short: "Stakes Bee nodes",
		Long:  `Stakes Bee nodes with BZZ tokens and ETH for Bee node operations.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: c.preRunE,
	}

	cmd.AddCommand(c.initStakeDeposit())

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) initStakeDeposit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit",
		Short: "Deposit stake on Bee nodes",
		Long:  `Deposits a specified amount of BZZ as stake on targeted Bee nodes.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			amount, err := cmd.Flags().GetString(optionNameAmount)
			if err != nil {
				return fmt.Errorf("error reading amount flag: %w", err)
			}

			if amount == "" {
				return errMissingStakeAmount
			}

			stakeAmount, ok := new(big.Int).SetString(amount, 10)
			if !ok {
				return errInvalidAmount
			}

			if stakeAmount.Cmp(big.NewInt(0)) <= 0 {
				return errInvalidAmount
			}

			fmt.Printf("Amount validated: %s WEI\n", amount)

			clusterName, err := cmd.Flags().GetString(optionNameClusterName)
			if err != nil {
				return fmt.Errorf("error reading cluster-name flag: %w", err)
			}

			if clusterName == "" {
				return fmt.Errorf("cluster-name is required")
			}

			fmt.Printf("Targeting cluster: %s\n", clusterName)

			// Setup cluster and get node clients
			ctx := context.Background()
			cluster, err := c.setupCluster(ctx, clusterName, false)
			if err != nil {
				return fmt.Errorf("failed to setup cluster %s: %w", clusterName, err)
			}

			clients, err := cluster.NodesClients(ctx)
			if err != nil {
				return fmt.Errorf("failed to get node clients: %w", err)
			}

			fmt.Printf("Found %d nodes in cluster\n", len(clients))

			fmt.Printf("Starting stake deposit of %s WEI on %d nodes...\n", amount, len(clients))

			var errorCount int
			for nodeName, client := range clients {
				fmt.Printf("Depositing stake on node %s...\n", nodeName)

				txHash, err := client.DepositStake(ctx, stakeAmount)
				if err != nil {
					errorCount++
					fmt.Printf("%s\n", fmt.Sprintf("node %s: %v", nodeName, err))
					continue
				}

				fmt.Printf("Successfully deposited stake on node %s, transaction: %s\n", nodeName, txHash)
			}

			if errorCount > 0 {
				return fmt.Errorf("stake deposit completed with %d errors", errorCount)
			}

			fmt.Printf("Stake deposit completed successfully on all %d nodes!\n", len(clients))
			return nil
		},
	}

	cmd.Flags().String(optionNameAmount, "", "Stake amount in WEI (required)")
	if err := cmd.MarkFlagRequired(optionNameAmount); err != nil {
		return nil
	}
	cmd.Flags().String(optionNameClusterName, "", "Target Beekeeper cluster name (required)")
	if err := cmd.MarkFlagRequired(optionNameClusterName); err != nil {
		return nil
	}

	return cmd
}

package cmd

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

const (
	optionNameAmount   = "amount"
	optionNameParallel = "parallel"
	maxParallel        = 10
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

			parallel, err := cmd.Flags().GetInt(optionNameParallel)
			if err != nil {
				fmt.Printf("Warning: Could not read parallel flag, using default value of 5\n")
				parallel = 5
			}

			if parallel <= 0 {
				fmt.Printf("Warning: Invalid parallel value (%d), using default value of 5\n", parallel)
				parallel = 5
			}

			if parallel > len(clients) {
				fmt.Printf("Info: Parallel value (%d) is greater than number of nodes (%d), using %d\n", parallel, len(clients), len(clients))
				parallel = len(clients)
			}

			// Cap parallel operations to prevent network overload
			if parallel > maxParallel {
				fmt.Printf("Info: Parallel value (%d) is too high, capping at %d to prevent network overload\n", parallel, maxParallel)
				parallel = maxParallel
			}

			fmt.Printf("Starting stake deposit of %s WEI on %d nodes with %d parallel operations...\n", amount, len(clients), parallel)

			var errorCount int
			var mu sync.Mutex
			semaphore := make(chan struct{}, parallel)
			var wg sync.WaitGroup

			for nodeName, client := range clients {
				wg.Add(1)
				go func(name string, cl *bee.Client) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					fmt.Printf("Depositing stake on node %s...\n", name)

					txHash, err := cl.DepositStake(ctx, stakeAmount)
					if err != nil {
						mu.Lock()
						errorCount++
						mu.Unlock()
						fmt.Printf("%s\n", c.formatStakeError(name, err))
						return
					}

					fmt.Printf("Successfully deposited stake on node %s, transaction: %s\n", name, txHash)
				}(nodeName, client)
			}

			wg.Wait()

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
	cmd.Flags().Int(optionNameParallel, 5, "Number of parallel operations (default: 5, max: number of nodes)")

	return cmd
}

// formatStakeError formats stake-related errors with user-friendly messages
func (c *command) formatStakeError(nodeName string, err error) string {
	errorStr := err.Error()

	if strings.Contains(errorStr, "out of funds") {
		return fmt.Sprintf("node %s: insufficient BZZ balance (fund the node wallet first)", nodeName)
	} else if strings.Contains(errorStr, "insufficient stake amount") {
		return fmt.Sprintf("node %s: stake amount too low (increase the amount)", nodeName)
	} else if strings.Contains(errorStr, "503") {
		return fmt.Sprintf("node %s: service temporarily unavailable (node might be starting up)", nodeName)
	} else {
		return fmt.Sprintf("node %s: %v", nodeName, err)
	}
}

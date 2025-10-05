package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	nodefunder "github.com/ethersphere/beekeeper/pkg/funder/node"
	"github.com/ethersphere/node-funder/pkg/funder"
	"github.com/spf13/cobra"
)

const (
	nodeFunderLabelSelector string = "beekeeper.ethswarm.org/node-funder=true"
	nodeFunderCmd           string = "node-funder"
)

func (c *command) initNodeFunderCmd() (err error) {
	const (
		optionNameAddresses     = "addresses"
		optionNameNamespace     = "namespace"
		optionNameWalletKey     = "wallet-key"
		optionNameMinNative     = "min-native"
		optionNameMinSwarm      = "min-swarm"
		optionNameTimeout       = "timeout"
		optionNameLabelSelector = "label-selector"
	)

	cmd := &cobra.Command{
		Use:   nodeFunderCmd,
		Short: "Funds bee nodes with ETH and BZZ",
		Long: `Funds Bee nodes with ETH and BZZ tokens to maintain operational requirements.

The node-funder command automatically manages funding for Bee nodes in three ways:
• Fund specific addresses: Provide a list of Ethereum addresses to fund
• Fund by namespace: Target all nodes in a specific Kubernetes namespace
• Fund by cluster: Fund all nodes in a Beekeeper-managed cluster

The command ensures nodes maintain minimum balances for:
• Native coins (xDAI) for gas fees and transactions
• Swarm tokens (xBZZ) for postage and network operations

Use --periodic-check to set up continuous funding monitoring.
Use --label-selector to filter nodes within a namespace.
Requires --wallet-key for the funding account and --geth-url for blockchain access.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				cfg := funder.Config{
					MinAmounts: funder.MinAmounts{
						NativeCoin: c.globalConfig.GetFloat64(optionNameMinNative),
						SwarmToken: c.globalConfig.GetFloat64(optionNameMinSwarm),
					},
				}

				if cfg.ChainNodeEndpoint = c.globalConfig.GetString(optionNameGethURL); cfg.ChainNodeEndpoint == "" {
					return errBlockchainEndpointNotProvided
				}

				if cfg.WalletKey = c.globalConfig.GetString(optionNameWalletKey); cfg.WalletKey == "" {
					return errors.New("wallet key not provided")
				}

				defer c.log.Infof("node-funder done")

				logOpt := funder.WithLoggerOption(c.log)

				addresses := c.globalConfig.GetStringSlice(optionNameAddresses)
				if len(addresses) > 0 {
					cfg.Addresses = addresses
					return c.executePeriodically(ctx, func(ctx context.Context) error {
						return funder.Fund(ctx, cfg, nil, nil, logOpt)
					})
				}

				nodeClient, err := c.createNodeClient(ctx, false)
				if err != nil {
					return fmt.Errorf("creating node client: %w", err)
				}

				if c.globalConfig.IsSet(optionNameNamespace) {
					cfg.Namespace = nodeClient.Namespace()
					return c.executePeriodically(ctx, func(ctx context.Context) error {
						funderClient := nodefunder.NewClient(nodeClient, c.log)
						return funder.Fund(ctx, cfg, funderClient, nil, logOpt)
					})
				}

				if c.globalConfig.IsSet(optionNameClusterName) {
					nodes, err := nodeClient.GetNodes(ctx)
					if err != nil {
						return fmt.Errorf("failed to retrieve nodes: %w", err)
					}

					for _, node := range nodes {
						addr, err := node.Client().Node.Addresses(ctx)
						if err != nil {
							return fmt.Errorf("error fetching addresses for node %s: %w", node.Name(), err)
						}
						cfg.Addresses = append(cfg.Addresses, addr.Ethereum)
					}

					return c.executePeriodically(ctx, func(ctx context.Context) error {
						return funder.Fund(ctx, cfg, nil, nil, logOpt)
					})
				}

				// NOTE: Swarm key address is the same as the nodeEndpoint/wallet walletAddress.
				// When setting up a bootnode, the swarmkey option is used to specify the existing swarm key.
				// However, for other nodes, the beekeeper automatically generates a new swarm key during cluster setup.
				// Once the swarm key is generated, beekeeper identifies the addresses that can be funded for each node.

				return errors.New("one of addresses, namespace, or valid cluster-name must be provided")
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringSliceP(optionNameAddresses, "a", nil, "Comma-separated list of Bee node addresses (must start with 0x). Overrides namespace and cluster name.")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace. Overrides cluster name if set.")
	cmd.Flags().String(optionNameClusterName, "", "Name of the Beekeeper cluster to target. Ignored if a namespace is specified.")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Float64(optionNameMinNative, 0, "Minimum amount of chain native coins (xDAI) nodes should have.")
	cmd.Flags().Float64(optionNameMinSwarm, 0, "Minimum amount of swarm tokens (xBZZ) nodes should have.")
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the specified namespace. Use an empty string to select all resources.")
	cmd.Flags().StringSlice(optionNameNodeGroups, nil, "List of node groups to target for node-funder (applies to all groups if not set). Only used with --cluster-name.")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Operation timeout (e.g., 5s, 10m, 1.5h).")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0*time.Minute, "Periodic execution check interval.")

	c.root.AddCommand(cmd)

	return nil
}

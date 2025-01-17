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
		optionNameAddresses         = "addresses"
		optionNameNamespace         = "namespace"
		optionNameChainNodeEndpoint = "geth-url"
		optionNameWalletKey         = "wallet-key"
		optionNameMinNative         = "min-native"
		optionNameMinSwarm          = "min-swarm"
		optionNameTimeout           = "timeout"
		optionNameLabelSelector     = "label-selector"
	)

	cmd := &cobra.Command{
		Use:   nodeFunderCmd,
		Short: "funds bee nodes with ETH and BZZ",
		Long:  `Fund makes BZZ tokens and ETH deposits to given Ethereum addresses. beekeeper node-funder`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := funder.Config{
				MinAmounts: funder.MinAmounts{
					NativeCoin: c.globalConfig.GetFloat64(optionNameMinNative),
					SwarmToken: c.globalConfig.GetFloat64(optionNameMinSwarm),
				},
			}

			// chain node endpoint check
			if cfg.ChainNodeEndpoint = c.globalConfig.GetString(optionNameChainNodeEndpoint); cfg.ChainNodeEndpoint == "" {
				return errors.New("chain node endpoint (geth-url) not provided")
			}

			// wallet key check
			if cfg.WalletKey = c.globalConfig.GetString(optionNameWalletKey); cfg.WalletKey == "" {
				return errors.New("wallet key not provided")
			}

			// add timeout to node-funder
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()
			defer c.log.Infof("node-funder done")

			logger := funder.WithLoggerOption(c.log)

			addresses := c.globalConfig.GetStringSlice(optionNameAddresses)
			if len(addresses) > 0 {
				cfg.Addresses = addresses
				return c.executePeriodically(ctx, func(ctx context.Context) error {
					return funder.Fund(ctx, cfg, nil, nil, logger)
				})
			}

			namespace := c.globalConfig.GetString(optionNameNamespace)
			if namespace != "" {
				label := c.globalConfig.GetString(optionNameLabelSelector)
				funderClient := nodefunder.NewClient(c.k8sClient, c.globalConfig.GetBool(optionNameInCluster), label, c.log)

				cfg.Namespace = namespace
				return c.executePeriodically(ctx, func(ctx context.Context) error {
					return funder.Fund(ctx, cfg, funderClient, nil, logger)
				})
			}

			clusterName := c.globalConfig.GetString(optionNameClusterName)
			if clusterName != "" {
				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
				}

				clients, err := cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to retrieve node clients: %w", err)
				}

				for _, node := range clients {
					addr, err := node.Addresses(ctx)
					if err != nil {
						return fmt.Errorf("error fetching addresses for node %s: %w", node.Name(), err)
					}
					cfg.Addresses = append(cfg.Addresses, addr.Ethereum)
				}

				return c.executePeriodically(ctx, func(ctx context.Context) error {
					return funder.Fund(ctx, cfg, nil, nil, logger)
				})
			}

			// NOTE: Swarm key address is the same as the nodeEndpoint/wallet walletAddress.
			// When setting up a bootnode, the swarmkey option is used to specify the existing swarm key.
			// However, for other nodes, the beekeeper automatically generates a new swarm key during cluster setup.
			// Once the swarm key is generated, beekeeper identifies the addresses that can be funded for each node.

			return errors.New("one of addresses, namespace, or valid cluster-name must be provided")
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringSliceP(optionNameAddresses, "a", nil, "Comma-separated list of Bee node addresses (must start with 0x). Overrides namespace and cluster name.")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace. Overrides cluster name if set.")
	cmd.Flags().String(optionNameClusterName, "", "Name of the Beekeeper cluster to target. Ignored if a namespace is specified.")
	cmd.Flags().String(optionNameChainNodeEndpoint, "", "Endpoint to chain node. Required.")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Float64(optionNameMinNative, 0, "Minimum amount of chain native coins (xDAI) nodes should have.")
	cmd.Flags().Float64(optionNameMinSwarm, 0, "Minimum amount of swarm tokens (xBZZ) nodes should have.")
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the specified namespace. Use an empty string to select all resources.")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Timeout.")
	cmd.Flags().Duration(optionNamePeriodicCheck, 0*time.Minute, "Periodic execution check interval.")

	c.root.AddCommand(cmd)

	return nil
}

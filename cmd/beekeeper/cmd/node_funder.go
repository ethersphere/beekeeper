package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/k8s"
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

			// add timeout to node-funder
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			c.logger.Infof("node-funder started")
			defer c.logger.Infof("node-funder done")

			return funder.Fund(ctx, funder.Config{
				Namespace:         cfg.Namespace,
				Addresses:         cfg.Addresses,
				ChainNodeEndpoint: cfg.ChainNodeEndpoint,
				WalletKey:         cfg.WalletKey,
				MinAmounts: funder.MinAmounts{
					NativeCoin: cfg.MinAmounts.NativeCoin,
					SwarmToken: cfg.MinAmounts.SwarmToken,
				},
			}, newNodeFunder(c.k8sClient))
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

type nodeFunder struct {
	k8sClient *k8s.Client
}

func newNodeFunder(k8sClient *k8s.Client) *nodeFunder {
	return &nodeFunder{
		k8sClient: k8sClient,
	}
}

func (nf *nodeFunder) List(ctx context.Context, namespace string) (nodes []funder.NodeInfo, err error) {
	if nf.k8sClient == nil {
		return nil, fmt.Errorf("k8s client not initialized")
	}

	if namespace == "" {
		return nil, fmt.Errorf("namespace not provided")
	}

	ingressHosts, err := nf.k8sClient.Ingress.ListDebugNodesHosts(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("list ingress debug nodes hosts: %s", err.Error())
	}

	ingressRouteHosts, err := nf.k8sClient.IngressRoute.ListDebugNodesHosts(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("list ingress route debug nodes hosts: %s", err.Error())
	}

	nodes = make([]funder.NodeInfo, 0, len(ingressHosts)+len(ingressRouteHosts))

	for _, node := range ingressHosts {
		nodes = append(nodes, funder.NodeInfo{
			Name:    node.Name,
			Address: node.Host,
		})
	}

	for _, node := range ingressRouteHosts {
		nodes = append(nodes, funder.NodeInfo{
			Name:    node.Name,
			Address: node.Host,
		})
	}

	return nodes, nil
}

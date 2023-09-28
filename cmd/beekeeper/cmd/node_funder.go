package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/node-funder/pkg/funder"
	"github.com/spf13/cobra"
)

func (c *command) initNodeFunderCmd() (err error) {
	const (
		optionNameAddresses         = "addresses"
		optionNameNamespace         = "namespace"
		optionClusterName           = "cluster-name"
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

			namespace := c.globalConfig.GetString(optionNameNamespace)
			addresses := c.globalConfig.GetStringSlice(optionNameAddresses)
			clusterName := c.globalConfig.GetString(optionClusterName)

			if len(addresses) > 0 {
				cfg.Addresses = addresses
			} else if namespace != "" {
				cfg.Namespace = namespace
			} else if clusterName != "" {
				cluster, ok := c.config.Clusters[clusterName]
				if !ok {
					return fmt.Errorf("cluster %s not found", clusterName)
				}
				if cluster.Namespace == nil || *cluster.Namespace == "" {
					return fmt.Errorf("cluster %s namespace not provided", clusterName)
				}
				cfg.Namespace = *cluster.Namespace
			} else {
				return errors.New("one of addresses, namespace, or valid cluster-name must be provided")
			}

			// chain node endpoint check
			if cfg.ChainNodeEndpoint = c.globalConfig.GetString(optionNameChainNodeEndpoint); cfg.ChainNodeEndpoint == "" {
				return errors.New("chain node endpoint not provided")
			}

			// wallet key check
			if cfg.WalletKey = c.globalConfig.GetString(optionNameWalletKey); cfg.WalletKey == "" {
				return errors.New("wallet key not provided")
			}

			cfg.MinAmounts.NativeCoin = c.globalConfig.GetFloat64(optionNameMinNative)
			cfg.MinAmounts.SwarmToken = c.globalConfig.GetFloat64(optionNameMinSwarm)

			// add timeout to node-funder
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			c.log.Infof("node-funder started")
			defer c.log.Infof("node-funder done")

			// TODO: Note that the swarm key address is the same as the nodeEndpoint/wallet walletAddress.

			// TODO: When setting up a bootnode, the swarmkey option is used to specify the existing swarm key.
			// However, for other nodes, the beekeeper automatically generates a new swarm key during cluster setup.
			// Once the swarm key is generated, beekeeper identifies the addresses that can be funded for each node.

			var nodeLister funder.NodeLister
			// if addresses are provided, use them, not k8s client to list nodes
			if cfg.Namespace != "" {
				nodeLister = newNodeLister(c.k8sClient, c.log)
			}

			return funder.Fund(ctx, funder.Config{
				Namespace:         cfg.Namespace,
				Addresses:         cfg.Addresses,
				ChainNodeEndpoint: cfg.ChainNodeEndpoint,
				WalletKey:         cfg.WalletKey,
				MinAmounts: funder.MinAmounts{
					NativeCoin: cfg.MinAmounts.NativeCoin,
					SwarmToken: cfg.MinAmounts.SwarmToken,
				},
			}, nodeLister, nil)
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringSliceP(optionNameAddresses, "a", nil, "Comma-separated list of Bee node addresses (must start with 0x). Overrides namespace and cluster name.")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace. Overrides cluster name if set.")
	cmd.Flags().String(optionClusterName, "", "Cluster name. Ignored if addresses or namespace are set.")
	cmd.Flags().String(optionNameChainNodeEndpoint, "", "Endpoint to chain node. Required.")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Float64(optionNameMinNative, 0, "Minimum amount of chain native coins (xDAI) nodes should have.")
	cmd.Flags().Float64(optionNameMinSwarm, 0, "Minimum amount of swarm tokens (xBZZ) nodes should have.")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Timeout.")

	c.root.AddCommand(cmd)

	return nil
}

type nodeLister struct {
	k8sClient *k8s.Client
	log       logging.Logger
}

func newNodeLister(k8sClient *k8s.Client, l logging.Logger) *nodeLister {
	return &nodeLister{
		k8sClient: k8sClient,
		log:       l,
	}
}

func (nf *nodeLister) List(ctx context.Context, namespace string) (nodes []funder.NodeInfo, err error) {
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

	ingressHosts = append(ingressHosts, ingressRouteHosts...)

	for _, node := range ingressHosts {
		nodes = append(nodes, funder.NodeInfo{
			Name:    node.Name,
			Address: fmt.Sprintf("http://%s", node.Host),
		})
	}

	return nodes, nil
}

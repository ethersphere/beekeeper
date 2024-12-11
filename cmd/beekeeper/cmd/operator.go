package cmd

import (
	"context"
	"errors"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/funder/operator"
	"github.com/spf13/cobra"
)

func (c *command) initOperatorCmd() (err error) {
	const (
		optionNameNamespace         = "namespace"
		optionNameChainNodeEndpoint = "geth-url"
		optionNameWalletKey         = "wallet-key"
		optionNameMinNative         = "min-native"
		optionNameMinSwarm          = "min-swarm"
		optionNameTimeout           = "timeout"
		optionNameLabelSelector     = "label-selector"
	)

	cmd := &cobra.Command{
		Use:   "node-operator",
		Short: "scans for scheduled pods and funds them",
		Long:  `Node operator scans for scheduled pods and funds them using node-funder. beekeeper node-operator`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.NodeFunder{}

			var namespace string
			if namespace = c.globalConfig.GetString(optionNameNamespace); namespace == "" {
				return errors.New("namespace not provided")
			}

			// chain node endpoint check
			if cfg.ChainNodeEndpoint = c.globalConfig.GetString(optionNameChainNodeEndpoint); cfg.ChainNodeEndpoint == "" {
				return errors.New("chain node endpoint (geth-url) not provided")
			}

			// wallet key check
			if cfg.WalletKey = c.globalConfig.GetString(optionNameWalletKey); cfg.WalletKey == "" {
				return errors.New("wallet key not provided")
			}

			cfg.MinAmounts.NativeCoin = c.globalConfig.GetFloat64(optionNameMinNative)
			cfg.MinAmounts.SwarmToken = c.globalConfig.GetFloat64(optionNameMinSwarm)

			// add timeout to operator
			// if timeout is not set, operator will run infinitely
			var ctxNew context.Context
			var cancel context.CancelFunc
			timeout := c.globalConfig.GetDuration(optionNameTimeout)
			if timeout > 0 {
				ctxNew, cancel = context.WithTimeout(cmd.Context(), timeout)
			} else {
				ctxNew = context.Background()
			}
			if cancel != nil {
				defer cancel()
			}

			return operator.NewClient(&operator.ClientConfig{
				Log:               c.log,
				Namespace:         namespace,
				WalletKey:         cfg.WalletKey,
				ChainNodeEndpoint: cfg.ChainNodeEndpoint,
				MinAmounts:        cfg.MinAmounts,
				K8sClient:         c.k8sClient,
				LabelSelector:     c.globalConfig.GetString(optionNameLabelSelector),
			}).Run(ctxNew)
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace to scan for scheduled pods.")
	cmd.Flags().String(optionNameChainNodeEndpoint, "", "Endpoint to chain node. Required.")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Float64(optionNameMinNative, 0, "Minimum amount of chain native coins (xDAI) nodes should have.")
	cmd.Flags().Float64(optionNameMinSwarm, 0, "Minimum amount of swarm tokens (xBZZ) nodes should have.")
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the specified namespace. An empty string disables filtering, allowing all resources to be selected.")
	cmd.Flags().Duration(optionNameTimeout, 0*time.Minute, "Timeout. Default is infinite.")

	c.root.AddCommand(cmd)

	return nil
}

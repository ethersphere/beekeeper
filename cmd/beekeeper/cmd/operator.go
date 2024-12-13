package cmd

import (
	"context"
	"errors"
	"time"

	"github.com/ethersphere/beekeeper/pkg/funder/operator"
	"github.com/spf13/cobra"
)

const nodeOperatorCmd string = "node-operator"

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
		Use:   nodeOperatorCmd,
		Short: "scans for scheduled pods and funds them",
		Long:  `Node operator scans for scheduled pods and funds them using node-funder. beekeeper node-operator`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var namespace string
			if namespace = c.globalConfig.GetString(optionNameNamespace); namespace == "" {
				return errors.New("namespace not provided")
			}

			// chain node endpoint check
			var chainNodeEndpoint string
			if chainNodeEndpoint = c.globalConfig.GetString(optionNameChainNodeEndpoint); chainNodeEndpoint == "" {
				return errors.New("chain node endpoint (geth-url) not provided")
			}

			// wallet key check
			var walletKey string
			if walletKey = c.globalConfig.GetString(optionNameWalletKey); walletKey == "" {
				return errors.New("wallet key not provided")
			}

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
				WalletKey:         walletKey,
				ChainNodeEndpoint: chainNodeEndpoint,
				NativeToken:       c.globalConfig.GetFloat64(optionNameMinNative),
				SwarmToken:        c.globalConfig.GetFloat64(optionNameMinSwarm),
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
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the specified namespace. Use an empty string to select all resources.")
	cmd.Flags().Duration(optionNameTimeout, 0*time.Minute, "Maximum duration to wait for the operation to complete. Default is no timeout.")

	c.root.AddCommand(cmd)

	return nil
}

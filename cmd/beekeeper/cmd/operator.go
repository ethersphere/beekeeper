package cmd

import (
	"context"
	"errors"
	"time"

	"github.com/ethersphere/beekeeper/pkg/funder/operator"
	"github.com/spf13/cobra"
)

func (c *command) initOperatorCmd() error {
	const (
		optionNameNamespace     = "namespace"
		optionNameWalletKey     = "wallet-key"
		optionNameMinNative     = "min-native"
		optionNameMinSwarm      = "min-swarm"
		optionNameTimeout       = "timeout"
		optionNameLabelSelector = "label-selector"
	)

	cmd := &cobra.Command{
		Use:   "node-operator",
		Short: "Scans for scheduled Kubernetes pods and funds them",
		Long: `Automatically scans for scheduled Kubernetes pods and funds them using the node-funder system.

The node-operator command runs continuously in a Kubernetes namespace, monitoring for:
• New Bee node deployments that need initial funding
• Existing nodes that require balance top-ups
• Pod scheduling events that trigger funding operations

This operator ensures that all Bee nodes in the namespace maintain sufficient balances for:
• Native coins (xDAI) for gas fees and transactions
• Swarm tokens (xBZZ) for postage and network operations

The operator uses the "app.kubernetes.io/name=bee" label by default to identify Bee nodes,
but this can be customized with --label-selector. Runs indefinitely until manually stopped.

Requires --namespace, --wallet-key, and --geth-url for operation.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				namespace := c.globalConfig.GetString(optionNameNamespace)
				if namespace == "" {
					return errors.New("namespace not provided")
				}

				if !c.globalConfig.IsSet(optionNameGethURL) {
					return errBlockchainEndpointNotProvided
				}

				walletKey := c.globalConfig.GetString(optionNameWalletKey)
				if walletKey == "" {
					return errors.New("wallet key not provided")
				}

				return operator.NewClient(&operator.ClientConfig{
					Log:               c.log,
					Namespace:         namespace,
					WalletKey:         walletKey,
					ChainNodeEndpoint: c.globalConfig.GetString(optionNameGethURL),
					NativeToken:       c.globalConfig.GetFloat64(optionNameMinNative),
					SwarmToken:        c.globalConfig.GetFloat64(optionNameMinSwarm),
					K8sClient:         c.k8sClient,
					LabelSelector:     c.globalConfig.GetString(optionNameLabelSelector),
				}).Run(ctx)
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace to scan for scheduled pods.")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Float64(optionNameMinNative, 0, "Minimum amount of chain native coins (xDAI) nodes should have.")
	cmd.Flags().Float64(optionNameMinSwarm, 0, "Minimum amount of swarm tokens (xBZZ) nodes should have.")
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the specified namespace. Use an empty string to select all resources.")
	cmd.Flags().Duration(optionNameTimeout, 0*time.Minute, "Operation timeout (e.g., 5s, 10m, 1.5h). Default is 0, which means no timeout.")

	c.root.AddCommand(cmd)

	return nil
}

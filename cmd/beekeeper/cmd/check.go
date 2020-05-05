package cmd

import (
	"github.com/spf13/cobra"
)

const (
	optionNameAPIHostnamePattern      = "api-hostnames"
	optionNameAPIDomain               = "api-domain"
	optionNameDebugAPIHostnamePattern = "debug-api-hostnames"
	optionNameDebugAPIDomain          = "debug-api-domain"
	optionNameNamespace               = "namespace"
	optionNameNodeCount               = "node-count"
)

func (c *command) initCheckCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run tests on Bee node(s)",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().String(optionNameAPIHostnamePattern, "bee-%d", "API hostname pattern")
	cmd.PersistentFlags().String(optionNameAPIDomain, "core.internal", "API DNS domain")
	cmd.PersistentFlags().String(optionNameDebugAPIHostnamePattern, "bee-%d-debug", "Debug API hostname pattern")
	cmd.PersistentFlags().String(optionNameDebugAPIDomain, "core.internal", "Debug API DNS domain")
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace")
	cmd.PersistentFlags().IntP(optionNameNodeCount, "c", 1, "node count")

	cmd.AddCommand(c.initCheckPeerCount())
	cmd.AddCommand(c.initCheckPingPong())
	cmd.AddCommand(c.initCheckPushSync())

	c.root.AddCommand(cmd)
	return nil
}

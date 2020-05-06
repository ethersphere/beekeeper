package cmd

import (
	"github.com/spf13/cobra"
)

const (
	optionNameAPIHostnamePattern      = "api-hostnames"
	optionNameAPIDomain               = "api-domain"
	optionNameDebugAPIHostnamePattern = "debug-api-hostnames"
	optionNameDebugAPIDomain          = "debug-api-domain"
	optionNameDisableNamespace        = "disable-namespace"
	optionNameNamespace               = "namespace"
	optionNameNodeCount               = "node-count"
)

var disableNamespace bool

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
	cmd.PersistentFlags().String(optionNameDebugAPIHostnamePattern, "bee-%d-debug", "debug API hostname pattern")
	cmd.PersistentFlags().String(optionNameDebugAPIDomain, "core.internal", "debug API DNS domain")
	cmd.PersistentFlags().BoolVar(&disableNamespace, optionNameDisableNamespace, false, "disable Kubernetes namespace")
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace, must be set or disabled")
	cmd.PersistentFlags().IntP(optionNameNodeCount, "c", 1, "node count")

	cmd.AddCommand(c.initCheckFullConnectivity())
	cmd.AddCommand(c.initCheckPeerCount())
	cmd.AddCommand(c.initCheckPingPong())
	cmd.AddCommand(c.initCheckPushSync())

	c.root.AddCommand(cmd)
	return nil
}

func (c *command) checkPreRunE(cmd *cobra.Command, args []string) (err error) {
	if !disableNamespace {
		if err = cmd.MarkFlagRequired(optionNameNamespace); err != nil {
			return
		}
	}
	if err = c.config.BindPFlags(cmd.Flags()); err != nil {
		return
	}
	if !disableNamespace && len(c.config.GetString(optionNameNamespace)) == 0 {
		return cmd.Help()
	}
	return
}

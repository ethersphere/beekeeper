package cmd

import (
	"github.com/spf13/cobra"
)

const (
	optionNameAPIScheme               = "api-scheme"
	optionNameAPIHostnamePattern      = "api-hostnames"
	optionNameAPIDomain               = "api-domain"
	optionNameAPIInsecureTLS          = "api-insecure-tls"
	optionNameDebugAPIScheme          = "debug-api-scheme"
	optionNameDebugAPIHostnamePattern = "debug-api-hostnames"
	optionNameDebugAPIDomain          = "debug-api-domain"
	optionNameDebugAPIInsecureTLS     = "debug-api-insecure-tls"
	optionNameDisableNamespace        = "disable-namespace"
	optionNameInsecureTLS             = "insecure-tls"
	optionNameNamespace               = "namespace"
	optionNameNodeCount               = "node-count"
)

var (
	disableNamespace    bool
	insecureTLSAPI      bool
	insecureTLSDebugAPI bool
)

func (c *command) initCheckCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run tests on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().String(optionNameAPIScheme, "https", "API scheme")
	cmd.PersistentFlags().String(optionNameAPIHostnamePattern, "bee-%d", "API hostname pattern")
	cmd.PersistentFlags().String(optionNameAPIDomain, "core.internal", "API DNS domain")
	cmd.PersistentFlags().BoolVar(&insecureTLSAPI, optionNameAPIInsecureTLS, false, "skips TLS verification for API")
	cmd.PersistentFlags().String(optionNameDebugAPIScheme, "https", "debug API scheme")
	cmd.PersistentFlags().String(optionNameDebugAPIHostnamePattern, "bee-%d-debug", "debug API hostname pattern")
	cmd.PersistentFlags().String(optionNameDebugAPIDomain, "core.internal", "debug API DNS domain")
	cmd.PersistentFlags().BoolVar(&insecureTLSDebugAPI, optionNameDebugAPIInsecureTLS, false, "skips TLS verification for debug API")
	cmd.PersistentFlags().BoolVar(&disableNamespace, optionNameDisableNamespace, false, "disable Kubernetes namespace")
	cmd.PersistentFlags().Bool(optionNameInsecureTLS, false, "skips TLS verification for both API and debug API")
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace, must be set or disabled")
	cmd.PersistentFlags().IntP(optionNameNodeCount, "c", 1, "node count")

	cmd.AddCommand(c.initCheckFullConnectivity())
	cmd.AddCommand(c.initCheckKademlia())
	cmd.AddCommand(c.initCheckPeerCount())
	cmd.AddCommand(c.initCheckPingPong())
	cmd.AddCommand(c.initCheckPushSync())
	cmd.AddCommand(c.initCheckRetrieval())

	c.root.AddCommand(cmd)
	return nil
}

func (c *command) checkPreRunE(cmd *cobra.Command, args []string) (err error) {
	if !disableNamespace && len(c.config.GetString(optionNameNamespace)) == 0 {
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

	if c.config.GetBool(optionNameInsecureTLS) {
		insecureTLSAPI = true
		insecureTLSDebugAPI = true
	}

	return
}

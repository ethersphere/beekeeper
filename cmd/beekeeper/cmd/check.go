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
	optionNamePushGateway             = "push-gateway"
	optionNamePushMetrics             = "push-metrics"
)

var (
	disableNamespace    bool
	insecureTLSAPI      bool
	insecureTLSDebugAPI bool
	pushMetrics         bool
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
	cmd.PersistentFlags().String(optionNameAPIDomain, "staging.internal", "API DNS domain")
	cmd.PersistentFlags().BoolVar(&insecureTLSAPI, optionNameAPIInsecureTLS, false, "skips TLS verification for API")
	cmd.PersistentFlags().String(optionNameDebugAPIScheme, "https", "debug API scheme")
	cmd.PersistentFlags().String(optionNameDebugAPIHostnamePattern, "bee-%d-debug", "debug API hostname pattern")
	cmd.PersistentFlags().String(optionNameDebugAPIDomain, "staging.internal", "debug API DNS domain")
	cmd.PersistentFlags().BoolVar(&insecureTLSDebugAPI, optionNameDebugAPIInsecureTLS, false, "skips TLS verification for debug API")
	cmd.PersistentFlags().BoolVar(&disableNamespace, optionNameDisableNamespace, false, "disable Kubernetes namespace")
	cmd.PersistentFlags().Bool(optionNameInsecureTLS, false, "skips TLS verification for both API and debug API")
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace, must be set or disabled")
	cmd.PersistentFlags().IntP(optionNameNodeCount, "c", 1, "node count")
	cmd.PersistentFlags().String(optionNamePushGateway, "http://localhost:9091/", "Prometheus PushGateway")
	cmd.PersistentFlags().BoolVar(&pushMetrics, optionNamePushMetrics, false, "push metrics to pushgateway")

	cmd.AddCommand(c.initCheckBalances())
	cmd.AddCommand(c.initCheckFileRetrieval())
	cmd.AddCommand(c.initCheckFileRetrievalDynamic())
	cmd.AddCommand(c.initCheckFullConnectivity())
	cmd.AddCommand(c.initCheckKademlia())
	cmd.AddCommand(c.initCheckLocalPinning())
	cmd.AddCommand(c.initCheckPeerCount())
	cmd.AddCommand(c.initCheckPingPong())
	cmd.AddCommand(c.initCheckPullSync())
	cmd.AddCommand(c.initCheckPushSync())
	cmd.AddCommand(c.initCheckRetrieval())
	cmd.AddCommand(c.initCheckChunkRepair())
	cmd.AddCommand(c.initCheckManifest())

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

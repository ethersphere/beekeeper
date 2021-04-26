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
	optionNameInCluster               = "in-cluster"
	optionNameKubeconfig              = "kubeconfig"
	optionNameNamespace               = "namespace"
	optionNameNodeCount               = "node-count"
	optionNamePushGateway             = "push-gateway"
	optionNamePushMetrics             = "push-metrics"
	optionNameStartCluster            = "start-cluster"
)

var (
	disableNamespace    bool
	inCluster           bool
	insecureTLSAPI      bool
	insecureTLSDebugAPI bool
	pushMetrics         bool
	startCluster        bool
)

func (c *command) initPrintCmd() (err error) {
	const (
		optionNameClusterName = "cluster-name"
	)
	var (
		clusterName string
	)

	cmd := &cobra.Command{
		Use:   "print",
		Short: "Print Bee cluster info",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
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
	cmd.PersistentFlags().String(optionNameKubeconfig, "", "kubernetes config file")

	cmd.AddCommand(c.initPrintAddresses())
	cmd.AddCommand(c.initPrintOverlay())
	cmd.AddCommand(c.initPrintPeers())
	cmd.AddCommand(c.initPrintTopologies())
	cmd.AddCommand(c.initPrintDepths())

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) printPreRunE(cmd *cobra.Command, args []string) (err error) {
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

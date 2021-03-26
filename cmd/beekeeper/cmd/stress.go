package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initStressCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "stress",
		Short: "Run stress on a Bee cluster",
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
	cmd.PersistentFlags().BoolVar(&inCluster, optionNameInCluster, false, "run Beekeeper in Kubernetes cluster")
	cmd.PersistentFlags().String(optionNameKubeconfig, "", "kubernetes config file")

	cmd.AddCommand(c.initStressUpload())

	c.root.AddCommand(cmd)
	return nil
}

func (c *command) stressPreRunE(cmd *cobra.Command, args []string) (err error) {
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

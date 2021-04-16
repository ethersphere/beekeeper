package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
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

func (c *command) initCheckCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run tests on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}

			for _, checkName := range cfg.Playbooks[cfg.Execute].Checks {
				checkProfile, ok := cfg.Checks[checkName]
				if !ok {
					return fmt.Errorf("check %s doesn't exist", checkName)
				}

				check, ok := Checks[checkProfile.Name]
				if !ok {
					return fmt.Errorf("check %s not implemented", checkProfile.Name)
				}

				cluster, err := setupCluster(cmd.Context(), cfg, startCluster)
				if err != nil {
					return fmt.Errorf("cluster setup: %w", err)
				}

				o, err := check.NewOptions(cfg, checkProfile)
				if err != nil {
					return fmt.Errorf("creating check %s options: %w", checkProfile.Name, err)
				}

				if err := check.NewCheck().Run(cmd.Context(), cluster, o); err != nil {
					return fmt.Errorf("running check %s: %w", checkProfile.Name, err)
				}
			}

			return nil
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
	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")

	c.root.AddCommand(cmd)
	return nil
}

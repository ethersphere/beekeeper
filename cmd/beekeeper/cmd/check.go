package cmd

import (
	"time"

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
	optionNamePostageAmount           = "postage-amount"
	optionNamePostageDepth            = "postage-depth"
	optionNamePostageBatchhWait       = "postage-wait"
	optionNameCacheCapacity           = "cache-capacity"
)

var (
	disableNamespace    bool
	inCluster           bool
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
	cmd.PersistentFlags().BoolVar(&inCluster, optionNameInCluster, false, "run Beekeeper in Kubernetes cluster")
	cmd.PersistentFlags().String(optionNameKubeconfig, "", "kubernetes config file")
	cmd.PersistentFlags().Int64(optionNamePostageAmount, 1, "postage stamp amount")
	// CICD options
	cmd.PersistentFlags().BoolVar(&clefSignerEnable, optionNameClefSignerEnable, false, "enable Clef signer")
	cmd.PersistentFlags().Uint64Var(&dbCapacity, optionNameDBCapacity, 5000000, "DB capacity")
	cmd.PersistentFlags().Uint64Var(&paymentEarly, optionNamePaymentEarly, 100000000000, "payment early")
	cmd.PersistentFlags().Uint64Var(&paymentThreshold, optionNamePaymentThreshold, 1000000000000, "payment threshold")
	cmd.PersistentFlags().Uint64Var(&paymentTolerance, optionNamePaymentTolerance, 100000000000, "payment tolerance")
	cmd.PersistentFlags().BoolVar(&swapEnable, optionNameSwapEnable, false, "enable swap")
	cmd.PersistentFlags().StringVar(&swapEndpoint, optionNameSwapEndpoint, "ws://geth-swap.geth:8546", "swap endpoint")
	cmd.PersistentFlags().StringVar(&swapFactoryAddress, optionNameSwapFactoryAddress, "0x657241f4494a2f15ba75346e691d753a978c72df", "swap factory address")
	cmd.PersistentFlags().Uint64Var(&swapInitialDeposit, optionNameSwapInitialDeposit, 500000000000000000, "swap initial deposit")
	cmd.PersistentFlags().StringVar(&nodeSelector, optionNameNodeSelector, "bee-staging", "node selector")
	cmd.PersistentFlags().StringVar(&ingressClass, optionNameIngressClass, "nginx-internal", "ingress class")
	cmd.PersistentFlags().Duration(optionNamePostageBatchhWait, time.Second*5, "time to wait for batch to be mined")
	cmd.PersistentFlags().Int(optionNameCacheCapacity, 1000, "cache capacity in chunks")
	cmd.PersistentFlags().Uint64(optionNamePostageDepth, 16, "default depth for postage batches")

	cmd.AddCommand(c.initCheckBalances())
	cmd.AddCommand(c.initCheckFileRetrieval())
	cmd.AddCommand(c.initCheckFullConnectivity())
	cmd.AddCommand(c.initCheckKademlia())
	cmd.AddCommand(c.initCheckGc())
	cmd.AddCommand(c.initCheckLocalPinningChunk())
	cmd.AddCommand(c.initCheckLocalPinningBytes())
	cmd.AddCommand(c.initCheckLocalPinningRemote())
	cmd.AddCommand(c.initCheckPeerCount())
	cmd.AddCommand(c.initCheckPingPong())
	cmd.AddCommand(c.initCheckPullSync())
	cmd.AddCommand(c.initCheckPushSync())
	cmd.AddCommand(c.initCheckRetrieval())
	cmd.AddCommand(c.initCheckSettlements())
	cmd.AddCommand(c.initCheckCashout())
	cmd.AddCommand(c.initCheckChunkRepair())
	cmd.AddCommand(c.initCheckManifest())
	cmd.AddCommand(c.initCheckPing())
	cmd.AddCommand(c.initCheckSmoke())
	cmd.AddCommand(c.initCheckPSS())
	cmd.AddCommand(c.initCheckSOC())

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

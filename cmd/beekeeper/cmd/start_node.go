package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/spf13/cobra"
)

func (c *command) initStartNode() *cobra.Command {
	const (
		nodeName                  = "bee"
		nodeVersion               = "latest"
		optionNameStartStandalone = "standalone"
	)

	var (
		standalone bool
	)

	cmd := &cobra.Command{
		Use:   "node",
		Short: "Start Bee node",
		Long:  `Start Bee node.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx := context.Background()

			node := bee.NewClient(bee.ClientOptions{KubeconfigPath: c.config.GetString(optionNameStartKubeconfig)})

			namespace := c.config.GetString(optionNameStartNamespace)
			nodeConfig.Standalone = standalone

			k8sOptions := k8s.NodeStartOptions{
				Name:      nodeName,
				Namespace: namespace,
				Annotations: map[string]string{
					"createdBy": "beekeeper",
				},
				Labels: map[string]string{
					"app.kubernetes.io/name":       nodeName,
					"app.kubernetes.io/version":    nodeVersion,
					"app.kubernetes.io/managed-by": "beekeeper",
					"beekeeper/version":            beekeeper.Version,
				},
				Image:           fmt.Sprintf("ethersphere/bee:%s", nodeVersion),
				ImagePullPolicy: "Always",
				IngressAnnotations: map[string]string{
					"kubernetes.io/ingress.class":                        "nginx-internal",
					"nginx.ingress.kubernetes.io/affinity":               "cookie",
					"nginx.ingress.kubernetes.io/affinity-mode":          "persistent",
					"nginx.ingress.kubernetes.io/proxy-body-size":        "0",
					"nginx.ingress.kubernetes.io/proxy-read-timeout":     "7200",
					"nginx.ingress.kubernetes.io/proxy-send-timeout":     "7200",
					"nginx.ingress.kubernetes.io/session-cookie-max-age": "86400",
					"nginx.ingress.kubernetes.io/session-cookie-name":    "SWARMGATEWAY",
					"nginx.ingress.kubernetes.io/session-cookie-path":    "default",
					"nginx.ingress.kubernetes.io/ssl-redirect":           "true",
				},
				IngressClass: "nginx-internal",
				IngressHost:  "bee.beekeeper.staging.internal",
				LimitCPU:     "1",
				LimitMemory:  "2Gi",
				NodeSelector: map[string]string{
					"node-group": "bee-staging",
				},
				PodManagementPolicy: "OrderedReady",
				RestartPolicy:       "Always",
				RequestCPU:          "750m",
				RequestMemory:       "1Gi",
				Selector: map[string]string{
					"app.kubernetes.io/name":       "bee",
					"app.kubernetes.io/managed-by": "beekeeper",
				},
				UpdateStrategy: "OnDelete",
			}

			return node.Start(ctx, bee.StartOptions{
				Name:           nodeName,
				Version:        nodeVersion,
				Config:         nodeConfig,
				Implementation: k8sOptions,
			})
		},
		PreRunE: c.startPreRunE,
	}

	cmd.PersistentFlags().BoolVarP(&standalone, optionNameStartStandalone, "s", false, "start a standalone node")

	return cmd
}

var (
	nodeConfig = bee.Config{
		APIAddr:              ":8080",
		Bootnodes:            "/dns4/bee-0-headless.beekeeper.svc.cluster.local/tcp/7070/p2p/16Uiu2HAm6i4dFaJt584m2jubyvnieEECgqM2YMpQ9nusXfy8XFzL",
		ClefSignerEnable:     false,
		ClefSignerEndpoint:   "",
		CORSAllowedOrigins:   "",
		DataDir:              "/home/bee/.bee",
		DBCapacity:           5000000,
		DebugAPIAddr:         ":6060",
		DebugAPIEnable:       true,
		GatewayMode:          false,
		GlobalPinningEnabled: true,
		NATAddr:              "",
		NetworkID:            1987,
		P2PAddr:              ":7070",
		P2PQUICEnable:        false,
		P2PWSEnable:          false,
		Password:             "beekeeper",
		PasswordFile:         "",
		PaymentEarly:         10000,
		PaymentThreshold:     100000,
		PaymentTolerance:     10000,
		ResolverEndpoints:    "",
		Standalone:           false,
		SwapEnable:           false,
		SwapEndpoint:         "http://localhost:8545",
		SwapFactoryAddress:   "",
		SwapInitialDeposit:   100000000,
		TracingEnabled:       true,
		TracingEndpoint:      "jaeger-operator-jaeger-agent.observability:6831",
		TracingServiceName:   "bee",
		Verbosity:            5,
		WelcomeMessage:       "Welcome to the Swarm, you are Bee-ing connected!",
	}
)

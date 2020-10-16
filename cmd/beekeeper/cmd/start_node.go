package cmd

import (
	"context"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/bee"
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

			node := bee.NewClient(bee.ClientOptions{KubeconfigPath: c.config.GetString(optionNameStartConfig)})

			config.Standalone = standalone
			k8sOpts := bee.K8SOptions{
				Name:      nodeName,
				Namespace: c.config.GetString(optionNameStartNamespace),
				Annotations: map[string]string{
					"createdBy": "beekeeper",
				},
				Labels: map[string]string{
					"app.kubernetes.io/name":       nodeName,
					"app.kubernetes.io/version":    nodeVersion,
					"app.kubernetes.io/managed-by": "beekeeper",
					"beekeeper/version":            beekeeper.Version,
				},
			}

			return node.Start(ctx, bee.StartOptions{
				Name:    nodeName,
				Version: nodeVersion,
				Config:  config,
				K8S:     &k8sOpts,
			})
		},
		PreRunE: c.startPreRunE,
	}

	cmd.PersistentFlags().BoolVarP(&standalone, optionNameStartStandalone, "s", false, "start a standalone node")

	return cmd
}

var (
	config = bee.Config{
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

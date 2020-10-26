package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/bee"
	k8s "github.com/ethersphere/beekeeper/pkg/k8s/bee"
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
				Config:    nodeConfig,
				Name:      nodeName,
				Namespace: namespace,
				Annotations: map[string]string{
					"createdBy": "beekeeper",
				},
				ClefImage:           "ethersphere/clef:latest",
				ClefImagePullPolicy: "Always",
				ClefKey:             `{"address":"fd50ede4954655b993ed69238c55219da7e81acf","crypto":{"cipher":"aes-128-ctr","ciphertext":"1c0f603b0dffe53294c7ca02c1a2800d81d855970db0df1a84cc11bc1d6cf364","cipherparams":{"iv":"11c9ac512348d7ccfe5ee59d9c9388d3"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"f6d7a0947da105fa5ef70fa298f65409d12967108c0e6260f847dc2b10455b89"},"mac":"fc6585e300ad3cb21c5f648b16b8a59ca33bcf13c58197176ffee4786628eaeb"},"id":"4911f965-b425-4011-895d-a2008f859859","version":3}`,
				ClefPassword:        "clefbeesecret",
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
				IngressHost: "bee-0.beekeeper.staging.internal",
				IngressDebugAnnotations: map[string]string{
					"kubernetes.io/ingress.class": "nginx-internal",
				},
				IngressDebugHost: "bee-0-debug.beekeeper.staging.internal",
				LibP2PKey:        `{"address":"aa6675fb77f3f84304a00d5ea09902d8a500364091a457cf21e05a41875d48f7","crypto":{"cipher":"aes-128-ctr","ciphertext":"93effebd3f015f496367e14218cb26d22de8f899e1d7b7686deb6ab43c876ea5","cipherparams":{"iv":"627434462c2f960d37338022d27fc92e"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a59e72e725fe3de25dd9c55aa55a93ed0e9090b408065a7204e2f505653acb70"},"mac":"dfb1e7ad93252928a7ff21ea5b65e8a4b9bda2c2e09cb6a8ac337da7a3568b8c"},"version":3}`,
				LimitCPU:         "1",
				LimitMemory:      "2Gi",
				NodeSelector: map[string]string{
					"node-group": "bee-staging",
				},
				PersistenceEnabled:        false,
				PersistenceStorageClass:   "local-storage",
				PersistanceStorageRequest: "34Gi",
				PodManagementPolicy:       "OrderedReady",
				RestartPolicy:             "Always",
				RequestCPU:                "750m",
				RequestMemory:             "1Gi",
				Selector: map[string]string{
					"app.kubernetes.io/name":       "bee",
					"app.kubernetes.io/managed-by": "beekeeper",
				},
				SwarmKey:       `{"address":"f176839c150e52fe30e5c2b5c648465c6fdfa532","crypto":{"cipher":"aes-128-ctr","ciphertext":"352af096f0fca9dfbd20a6861bde43d988efe7f179e0a9ffd812a285fdcd63b9","cipherparams":{"iv":"613003f1f1bf93430c92629da33f8828"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"ad1d99a4c64c95c26131e079e8c8a82221d58bf66a7ceb767c33a4c376c564b8"},"mac":"cafda1bc8ca0ffc2b22eb69afd1cf5072fd09412243443be1b0c6832f57924b6"},"version":3}`,
				UpdateStrategy: "OnDelete",
			}

			return node.Start(ctx, bee.StartOptions{
				Name:    nodeName,
				Version: nodeVersion,
				Options: k8sOptions,
			})
		},
		PreRunE: c.startPreRunE,
	}

	cmd.PersistentFlags().BoolVarP(&standalone, optionNameStartStandalone, "s", false, "start a standalone node")

	return cmd
}

var (
	nodeConfig = k8s.Config{
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

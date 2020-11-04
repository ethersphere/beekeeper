package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

var (
	// defaultBeeConfig represents default Bee node configuration
	beeDefaultConfig = k8sBee.Config{
		APIAddr:              ":8080",
		Bootnodes:            "",
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
		TracingEnabled:       false,
		TracingEndpoint:      "jaeger-operator-jaeger-agent.observability:6831",
		TracingServiceName:   "bee",
		Verbosity:            5,
		WelcomeMessage:       "Welcome to the Swarm, you are Bee-ing connected!",
	}
	// defaultNodeGroupOptions represents default Bee node group options
	defaultNodeGroupOptions = bee.NodeGroupOptions{
		ClefImage:           "ethersphere/clef:latest",
		ClefImagePullPolicy: "Always",
		Image:               "ethersphere/bee:latest",
		ImagePullPolicy:     "Always",
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
		IngressDebugAnnotations: map[string]string{
			"kubernetes.io/ingress.class": "nginx-internal",
		},
		Labels: map[string]string{
			"app.kubernetes.io/component": "bee",
			"app.kubernetes.io/part-of":   "bee",
			"app.kubernetes.io/version":   "latest",
		},
		LimitCPU:    "1",
		LimitMemory: "2Gi",
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
		UpdateStrategy:            "RollingUpdate",
	}
)

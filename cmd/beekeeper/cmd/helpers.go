package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

// defaultBeeConfig returns default Bee node configuration
func defaultBeeConfig() *k8sBee.Config {
	return &k8sBee.Config{
		APIAddr:              ":1633",
		Bootnodes:            "",
		ClefSignerEnable:     false,
		ClefSignerEndpoint:   "",
		CORSAllowedOrigins:   "",
		DataDir:              "/home/bee/.bee",
		DBCapacity:           5000000,
		DebugAPIAddr:         ":1635",
		DebugAPIEnable:       true,
		GatewayMode:          false,
		GlobalPinningEnabled: true,
		NATAddr:              "",
		NetworkID:            1987,
		P2PAddr:              ":1634",
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
}

// defaultNodeGroupOptions returns default node group options
func defaultNodeGroupOptions() *bee.NodeGroupOptions {
	return &bee.NodeGroupOptions{
		ClefImage:           "ethersphere/clef:0.4.4",
		ClefImagePullPolicy: "IfNotPresent",
		Image:               "ethersphere/bee:0.4.1",
		ImagePullPolicy:     "IfNotPresent",
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
		PersistenceEnabled:        true,
		PersistenceStorageClass:   "local-storage",
		PersistanceStorageRequest: "34Gi",
		PodManagementPolicy:       "OrderedReady",
		RestartPolicy:             "Always",
		RequestCPU:                "750m",
		RequestMemory:             "1Gi",
		UpdateStrategy:            "RollingUpdate",
	}
}

type bootnodeSetup struct {
	Bootnodes    string
	ClefKey      string
	ClefPassword string
	LibP2PKey    string
	SwarmKey     string
}

func setupBootnodes(n int, ns string) []bootnodeSetup {
	switch n {
	case 2:
		return []bootnodeSetup{
			{
				Bootnodes:    fmt.Sprintf("/dns4/bootnode-1-headless.%s.svc.cluster.local/tcp/1634/p2p/16Uiu2HAmMw7Uj6vfraD9BYx3coDs6MK6pAmActE8fsfaZwigsaB6", ns),
				ClefKey:      `{"address":"fd50ede4954655b993ed69238c55219da7e81acf","crypto":{"cipher":"aes-128-ctr","ciphertext":"1c0f603b0dffe53294c7ca02c1a2800d81d855970db0df1a84cc11bc1d6cf364","cipherparams":{"iv":"11c9ac512348d7ccfe5ee59d9c9388d3"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"f6d7a0947da105fa5ef70fa298f65409d12967108c0e6260f847dc2b10455b89"},"mac":"fc6585e300ad3cb21c5f648b16b8a59ca33bcf13c58197176ffee4786628eaeb"},"id":"4911f965-b425-4011-895d-a2008f859859","version":3}`,
				ClefPassword: "clefbeesecret",
				LibP2PKey:    `{"address":"aa6675fb77f3f84304a00d5ea09902d8a500364091a457cf21e05a41875d48f7","crypto":{"cipher":"aes-128-ctr","ciphertext":"93effebd3f015f496367e14218cb26d22de8f899e1d7b7686deb6ab43c876ea5","cipherparams":{"iv":"627434462c2f960d37338022d27fc92e"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a59e72e725fe3de25dd9c55aa55a93ed0e9090b408065a7204e2f505653acb70"},"mac":"dfb1e7ad93252928a7ff21ea5b65e8a4b9bda2c2e09cb6a8ac337da7a3568b8c"},"version":3}`,
				SwarmKey:     `{"address":"f176839c150e52fe30e5c2b5c648465c6fdfa532","crypto":{"cipher":"aes-128-ctr","ciphertext":"352af096f0fca9dfbd20a6861bde43d988efe7f179e0a9ffd812a285fdcd63b9","cipherparams":{"iv":"613003f1f1bf93430c92629da33f8828"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"ad1d99a4c64c95c26131e079e8c8a82221d58bf66a7ceb767c33a4c376c564b8"},"mac":"cafda1bc8ca0ffc2b22eb69afd1cf5072fd09412243443be1b0c6832f57924b6"},"version":3}`,
			},
			{
				Bootnodes:    fmt.Sprintf("/dns4/bootnode-0-headless.%s.svc.cluster.local/tcp/1634/p2p/16Uiu2HAm6i4dFaJt584m2jubyvnieEECgqM2YMpQ9nusXfy8XFzL", ns),
				ClefKey:      `{"address":"804c5b6f71086bd2e9a74207995f0237ed43a39c","crypto":{"cipher":"aes-128-ctr","ciphertext":"a59325fd3896ed0ce19bd2a1878da5210ff1ca01ce0fd0800088448c4f120c95","cipherparams":{"iv":"be6116f2c1c1bc847f453f64c8a54c3d"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"9dc06f7bd249d1de4966a09235abf3b5f252eeb96130195105731b27b6d615d1"},"mac":"9534004532355f1f9f0fa5e7670fb5f687288aa191d65ec2351dffb6d7b1b80d"},"id":"dfa9d87c-6cf9-43db-a968-b53c3c3fbd8f","version":3}`,
				ClefPassword: "clefbeesecret",
				LibP2PKey:    `{"address":"03348ecf3adae1d05dc16e475a83c94e49e28a4d3c7db5eccbf5ca4ea7f688ddcdfe88acbebef2037c68030b1a0a367a561333e5c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470","crypto":{"cipher":"aes-128-ctr","ciphertext":"0d0ff25e9b03292e622c5a09ec00c2acb7ff5882f02dd2f00a26ac6d3292a434","cipherparams":{"iv":"cd4082caf63320b306fe885796ba224f"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a4d63d56c539eb3eff2a235090127486722fa2c836cf735d50d673b730cebc3f"},"mac":"aad40da9c1e742e2b01bb8f76ba99ace97ccb0539cea40e31eb6b9bb64a3f36a"},"version":3}`,
				SwarmKey:     `{"address":"ebe269e07161c68a942a3a7fce6b4ed66867d6f0","crypto":{"cipher":"aes-128-ctr","ciphertext":"06b550c35b46099aea8f6c9f799497d34bd5ebc13af79c7cdb2a1037227544ad","cipherparams":{"iv":"fa088e69b1849e40f190a5f69f0555f8"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"42b4f2815c0042d02eed916a7a74ecdc005f1f7eae0cfb5837c15f469df9ddba"},"mac":"23e3d0594ab94587258a33cc521edbde009b887a6f117ed7a3422d1c95123568"},"version":3}`,
			},
		}
	default:
		return []bootnodeSetup{{
			Bootnodes:    fmt.Sprintf("/dns4/bootnode-0-headless.%s.svc.cluster.local/tcp/1634/p2p/16Uiu2HAm6i4dFaJt584m2jubyvnieEECgqM2YMpQ9nusXfy8XFzL", ns),
			ClefKey:      `{"address":"fd50ede4954655b993ed69238c55219da7e81acf","crypto":{"cipher":"aes-128-ctr","ciphertext":"1c0f603b0dffe53294c7ca02c1a2800d81d855970db0df1a84cc11bc1d6cf364","cipherparams":{"iv":"11c9ac512348d7ccfe5ee59d9c9388d3"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"f6d7a0947da105fa5ef70fa298f65409d12967108c0e6260f847dc2b10455b89"},"mac":"fc6585e300ad3cb21c5f648b16b8a59ca33bcf13c58197176ffee4786628eaeb"},"id":"4911f965-b425-4011-895d-a2008f859859","version":3}`,
			ClefPassword: "clefbeesecret",
			LibP2PKey:    `{"address":"aa6675fb77f3f84304a00d5ea09902d8a500364091a457cf21e05a41875d48f7","crypto":{"cipher":"aes-128-ctr","ciphertext":"93effebd3f015f496367e14218cb26d22de8f899e1d7b7686deb6ab43c876ea5","cipherparams":{"iv":"627434462c2f960d37338022d27fc92e"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a59e72e725fe3de25dd9c55aa55a93ed0e9090b408065a7204e2f505653acb70"},"mac":"dfb1e7ad93252928a7ff21ea5b65e8a4b9bda2c2e09cb6a8ac337da7a3568b8c"},"version":3}`,
			SwarmKey:     `{"address":"f176839c150e52fe30e5c2b5c648465c6fdfa532","crypto":{"cipher":"aes-128-ctr","ciphertext":"352af096f0fca9dfbd20a6861bde43d988efe7f179e0a9ffd812a285fdcd63b9","cipherparams":{"iv":"613003f1f1bf93430c92629da33f8828"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"ad1d99a4c64c95c26131e079e8c8a82221d58bf66a7ceb767c33a4c376c564b8"},"mac":"cafda1bc8ca0ffc2b22eb69afd1cf5072fd09412243443be1b0c6832f57924b6"},"version":3}`,
		}}
	}
}

func setupBootnodesDNS(n int, ns string) string {
	switch n {
	case 2:
		return fmt.Sprintf("/dns4/bootnode-0-headless.%s.svc.cluster.local/tcp/1634/p2p/16Uiu2HAm6i4dFaJt584m2jubyvnieEECgqM2YMpQ9nusXfy8XFzL /dns4/bootnode-1-headless.%s.svc.cluster.local/tcp/1634/p2p/16Uiu2HAmMw7Uj6vfraD9BYx3coDs6MK6pAmActE8fsfaZwigsaB6", ns, ns)
	default:
		return fmt.Sprintf("/dns4/bootnode-0-headless.%s.svc.cluster.local/tcp/1634/p2p/16Uiu2HAm6i4dFaJt584m2jubyvnieEECgqM2YMpQ9nusXfy8XFzL", ns)
	}
}

func setK8SClient(kubeconfig string, inCluster bool) (c *k8s.Client, err error) {
	if c, err = k8s.NewClient(&k8s.ClientOptions{
		InCluster:      inCluster,
		KubeconfigPath: kubeconfig,
	}); err != nil {
		return nil, fmt.Errorf("creating Kubernetes client: %w", err)
	}

	return
}

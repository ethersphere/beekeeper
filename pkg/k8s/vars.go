package k8s

import (
	"github.com/ethersphere/beekeeper"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	name        = "bee"
	annotations = map[string]string{
		"createdBy": "beekeeper",
	}
	labels = map[string]string{
		"app.kubernetes.io/instance":   "bee",
		"app.kubernetes.io/managed-by": "beekeeper",
		"app.kubernetes.io/name":       "bee",
		"app.kubernetes.io/version":    "latest",
		"beekeeper/version":            beekeeper.Version,
	}
	cmData = map[string]string{
		".bee.yaml": `api-addr: :8080
bootnode: /dns4/bee-0-headless.svetomir.svc.cluster.local/tcp/7070/p2p/16Uiu2HAm6i4dFaJt584m2jubyvnieEECgqM2YMpQ9nusXfy8XFzL
clef-signer-enable: false
clef-signer-endpoint: 
cors-allowed-origins: 
data-dir: /home/bee/.bee
db-capacity: 5e+06
debug-api-addr: :6060
debug-api-enable: true
gateway-mode: false
global-pinning-enable: true
nat-addr: 
network-id: 4386
password: beekeeper
payment-threshold: 7000
payment-tolerance: 700
p2p-addr: :7070
p2p-quic-enable: false
p2p-ws-enable: false
resolver-options: 
standalone: false
swap-enable: false
swap-endpoint: http://localhost:8545
swap-factory-address: 
swap-initial-deposit: 0
tracing-enable: true
tracing-endpoint: jaeger-operator-jaeger-agent.observability:6831
tracing-service-name: bee
verbosity: 5
welcome-message: Welcome to the Swarm, you are Bee-ing connected!`,
	}
	svcSelector = map[string]string{
		"app.kubernetes.io/instance":   "bee",
		"app.kubernetes.io/name":       "bee",
		"app.kubernetes.io/managed-by": "beekeeper",
	}
	svcPorts = []v1.ServicePort{
		{
			Name:       "api",
			Protocol:   "TCP",
			Port:       80,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
		},
	}
	svcHeadlessPorts = []v1.ServicePort{
		{
			Name:       "api",
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
		},
		{
			Name:       "p2p",
			Protocol:   "TCP",
			Port:       7070,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "p2p"},
		},
		{
			Name:       "debug",
			Protocol:   "TCP",
			Port:       6060,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "debug"},
		},
	}
)

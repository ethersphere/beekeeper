package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	cmData = map[string]string{
		".bee.yaml": `api-addr: :8080
bootnode: /dns4/bee-0-headless.beekeeper.svc.cluster.local/tcp/7070/p2p/16Uiu2HAm6i4dFaJt584m2jubyvnieEECgqM2YMpQ9nusXfy8XFzL
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
network-id: 1987
password: beekeeper
payment-threshold: 100000
payment-tolerance: 10000
p2p-addr: :7070
p2p-quic-enable: false
p2p-ws-enable: false
resolver-options: 
standalone: true
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
)

// setConfigMap creates ConfigMap, if ConfigMap already exists updates in place
func setConfigMap(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, data map[string]string) (err error) {
	spec := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
			Labels:      labels,
		},
		Data: data,
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("configmap %s already exists in the namespace %s, updating the map\n", name, namespace)
			_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}

package k8s

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper"
	"k8s.io/client-go/kubernetes"
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
)

// Check ...
func Check(clientset *kubernetes.Clientset, namespace string) (err error) {
	ctx := context.Background()

	// namespace
	if err := setNamespace(ctx, clientset, namespace); err != nil {
		return fmt.Errorf("set namespace: %s", err)
	}

	// configuration
	if err := setConfigMap(ctx, clientset, namespace, name, cmData); err != nil {
		return fmt.Errorf("set configmap: %s", err)
	}

	// secrets
	if err := setSecret(ctx, clientset, namespace, fmt.Sprintf("%s-libp2p", name), secretData); err != nil {
		return fmt.Errorf("set secret: %s", err)
	}
	// services
	if err := setServiceAccount(ctx, clientset, namespace, name); err != nil {
		return fmt.Errorf("set serviceaccount %s", err)
	}

	if err := setService(ctx, clientset, namespace, name, svc); err != nil {
		return fmt.Errorf("set service %s", err)
	}

	if err := setService(ctx, clientset, namespace, fmt.Sprintf("%s-headless", name), svcHeadless); err != nil {
		return fmt.Errorf("set service %s", err)
	}

	// ingress
	if err := setIngress(ctx, clientset, namespace, name, ingressSpec); err != nil {
		return fmt.Errorf("set ingress %s", err)
	}

	// statefulset
	if err := setStatefulSet(ctx, clientset, namespace, fmt.Sprintf("%s-0", name), statefulsetSpec); err != nil {
		return fmt.Errorf("set statefulset %s", err)
	}

	return
}

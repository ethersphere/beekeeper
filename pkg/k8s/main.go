package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Check ...
func Check(clientset *kubernetes.Clientset, namespace string) (err error) {
	ctx := context.Background()

	if err := setNamespace(ctx, clientset, namespace); err != nil {
		return fmt.Errorf("set namespace: %s", err)
	}

	if err := setConfigMap(ctx, clientset, namespace, name, cmData); err != nil {
		return fmt.Errorf("set configmap: %s", err)
	}

	if err := setServiceAccount(ctx, clientset, namespace, name); err != nil {
		return fmt.Errorf("set serviceaccount %s", err)
	}

	if err := setService(ctx, clientset, namespace, name, v1.ServiceSpec{
		Ports:    svcPorts,
		Selector: svcSelector,
		Type:     v1.ServiceTypeClusterIP,
	}); err != nil {
		return fmt.Errorf("set service %s", err)
	}

	if err := setService(ctx, clientset, namespace, fmt.Sprintf("%s-headless", name), v1.ServiceSpec{
		Ports:    svcHeadlessPorts,
		Selector: svcSelector,
		Type:     v1.ServiceTypeClusterIP,
	}); err != nil {
		return fmt.Errorf("set service %s", err)
	}

	return
}

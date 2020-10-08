package k8s

import (
	"context"
	"fmt"

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

	return
}

package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

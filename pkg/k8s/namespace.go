package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// setNamespace creates namespace, if namespace already exists does nothing
func setNamespace(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (err error) {
	spec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespace,
			Annotations: annotations,
			Labels:      labels,
		},
	}

	_, err = clientset.CoreV1().Namespaces().Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("namespace %s already exists\n", namespace)
			return nil
		}
		return err
	}

	return
}

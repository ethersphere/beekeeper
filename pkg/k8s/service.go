package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// setService creates Service, if Service already exists does nothing
func setService(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, svcSpec v1.ServiceSpec) (err error) {
	spec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: svcSpec,
	}
	_, err = clientset.CoreV1().Services(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("service %s already exists in the namespace %s\n", name, namespace)
			return nil
		}

		return err
	}

	return
}

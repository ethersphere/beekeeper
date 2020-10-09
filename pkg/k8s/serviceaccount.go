package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// setServiceAccount creates ServiceAccount, if ServiceAccount already exists updates in place
func setServiceAccount(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (err error) {
	spec := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
			Labels:      labels,
		},
	}
	_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("service account %s already exists in the namespace %s, updating the service account\n", name, namespace)
			_, err = clientset.CoreV1().ServiceAccounts(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}

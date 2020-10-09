package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// setSecret creates Secret, if Secret already exists updates in place
func setSecret(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, strData map[string]string) (err error) {
	spec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
			Labels:      labels,
		},
		StringData: strData,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("secret %s already exists in the namespace %s, updating the secret\n", name, namespace)
			_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}

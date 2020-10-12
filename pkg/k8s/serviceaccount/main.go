package serviceaccount

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Options represents serviceaccount's options
type Options struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
}

// Set creates ServiceAccount, if ServiceAccount already exists updates in place
func Set(ctx context.Context, clientset *kubernetes.Clientset, o Options) (err error) {
	spec := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        o.Name,
			Namespace:   o.Namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
	}
	_, err = clientset.CoreV1().ServiceAccounts(o.Namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("service account %s already exists in the namespace %s, updating the service account\n", o.Name, o.Namespace)
			_, err = clientset.CoreV1().ServiceAccounts(o.Namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}

package namespace

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Service ...
type Service struct {
	clientset *kubernetes.Clientset
}

// NewService ...
func NewService(clientset *kubernetes.Clientset) *Service {
	return &Service{
		clientset: clientset,
	}
}

// Options represents namespace's options
type Options struct {
	Name        string
	Annotations map[string]string
	Labels      map[string]string
}

// Set creates namespace, if namespace already exists does nothing
func Set(ctx context.Context, clientset *kubernetes.Clientset, o Options) (err error) {
	spec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        o.Name,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
	}

	_, err = clientset.CoreV1().Namespaces().Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("namespace %s already exists, updating the namespace\n", o.Name)
			_, err = clientset.CoreV1().Namespaces().Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return nil
			}
		}
		return err
	}

	return
}

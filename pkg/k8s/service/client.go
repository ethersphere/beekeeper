package service

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Service.
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient constructs a new Client.
func NewClient(clientset *kubernetes.Clientset) *Client {
	return &Client{
		clientset: clientset,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	ServiceSpec Spec
}

// Set creates Service, if Service already exists does nothing
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: o.ServiceSpec.ToK8S(),
	}
	_, err = c.clientset.CoreV1().Services(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		// TODO: fix condition, always true
		if !errors.IsNotFound(err) {
			fmt.Printf("service %s already exists in the namespace %s\n", name, namespace)
			return nil
		}
		return
	}

	return
}

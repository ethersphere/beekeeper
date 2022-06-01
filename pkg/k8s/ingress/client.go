package ingress

import (
	"context"
	"fmt"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Ingress.
type Client struct {
	clientset kubernetes.Interface
}

// NewClient constructs a new Client.
func NewClient(clientset kubernetes.Interface) *Client {
	return &Client{
		clientset: clientset,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	Spec        Spec
}

// Set updates Ingress or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (ing *v1.Ingress, err error) {
	spec := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: o.Spec.toK8S(),
	}

	ing, err = c.clientset.NetworkingV1().Ingresses(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			ing, err = c.clientset.NetworkingV1().Ingresses(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("creating ingress %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return nil, fmt.Errorf("updating ingress %s in namespace %s: %w", name, namespace, err)
		}
	}

	return
}

// Delete deletes Ingress
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting ingress %s in namespace %s: %w", name, namespace, err)
	}

	return
}

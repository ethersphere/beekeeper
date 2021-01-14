package ingress

import (
	"context"
	"fmt"

	ev1b1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Ingress.
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
	Spec        Spec
}

// Set updates Ingress or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &ev1b1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: o.Spec.toK8S(),
	}

	_, err = c.clientset.ExtensionsV1beta1().Ingresses(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = c.clientset.ExtensionsV1beta1().Ingresses(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("creating ingress %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return fmt.Errorf("updating ingress %s in namespace %s: %w", name, namespace, err)
		}
	}

	return
}

// Delete deletes Ingress
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.ExtensionsV1beta1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting ingress %s in namespace %s: %w", name, namespace, err)
	}

	return
}

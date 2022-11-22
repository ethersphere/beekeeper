package ingressroute

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Client manages communication with the Kubernetes Ingress.
type Client struct {
	clientset Interface
}

// NewClient constructs a new Client.
func NewClient(clientset Interface) *Client {
	return &Client{
		clientset: clientset,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	Spec        IngressRouteSpec
}

// Set updates Ingress or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (ing *IngressRoute, err error) {
	spec := &IngressRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IngressRoute",
			APIVersion: GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: IngressRouteSpec{
			Routes: o.Spec.Routes,
		},
	}

	ing, err = c.clientset.IngressRoutes(namespace).Update(ctx, spec)
	if err != nil {
		if errors.IsNotFound(err) {
			ing, err = c.clientset.IngressRoutes(namespace).Create(ctx, spec)
			if err != nil {
				return nil, fmt.Errorf("creating ingress route %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return nil, fmt.Errorf("updating ingress route %s in namespace %s: %w", name, namespace, err)
		}
	}
	return
}

// Delete deletes IngressRoute
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.IngressRoutes(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting ingress route %s in namespace %s: %w", name, namespace, err)
	}

	return
}

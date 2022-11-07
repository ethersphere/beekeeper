package ingressroute

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// Client manages communication with the Kubernetes Ingress.
type Client struct {
	clientset Interface
}

// NewClient constructs a new Client.
func NewClient(clientset Interface) *Client {
	AddToScheme(scheme.Scheme)

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
	ing, err = c.clientset.IngressRoutes(namespace).Create(ctx, spec)
	return
}

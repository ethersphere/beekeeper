package ingressroute

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/k8s/customresource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Client manages communication with the Kubernetes Ingress.
type Client struct {
	clientset customresource.Interface
}

// NewClient constructs a new Client.
func NewClient(clientset customresource.Interface) *Client {
	return &Client{
		clientset: clientset,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	Spec        customresource.IngressRouteSpec
}

// Set updates Ingress or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (ing *customresource.IngressRoute, err error) {
	spec := &customresource.IngressRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IngressRoute",
			APIVersion: "traefik.containo.us/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: customresource.IngressRouteSpec{
			Routes: o.Spec.Routes,
		},
	}
	ing, err = c.clientset.IngressRoutes(namespace).Create(ctx, spec)
	return
}

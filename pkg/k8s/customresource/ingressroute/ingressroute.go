package ingressroute

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type IngressRouteInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*IngressRouteList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*IngressRoute, error)
	Create(ctx context.Context, ir *IngressRoute) (*IngressRoute, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type ingressRouteClient struct {
	restClient rest.Interface
	ns         string
}

const IngressRouteResource string = "ingressroutes"

func (c *ingressRouteClient) List(ctx context.Context, opts metav1.ListOptions) (*IngressRouteList, error) {
	result := IngressRouteList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(IngressRouteResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *ingressRouteClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*IngressRoute, error) {
	result := IngressRoute{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(IngressRouteResource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *ingressRouteClient) Create(ctx context.Context, ingressRoute *IngressRoute) (*IngressRoute, error) {
	result := IngressRoute{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource(IngressRouteResource).
		Body(ingressRoute).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *ingressRouteClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource(IngressRouteResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}

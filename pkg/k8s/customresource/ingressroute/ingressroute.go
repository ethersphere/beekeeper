package ingressroute

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// IngressRouteInterface has methods to work with IngressRoute resources.
type IngressRouteInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*IngressRouteList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*IngressRoute, error)
	Create(ctx context.Context, ir *IngressRoute) (*IngressRoute, error)
	Update(ctx context.Context, ir *IngressRoute, opts metav1.UpdateOptions) (*IngressRoute, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

// ingressRouteClient implements IngressRouteInterface.
type ingressRouteClient struct {
	restClient rest.Interface
	ns         string
}

const IngressRouteResource string = "ingressroutes"

// List takes label and field selectors, and returns the list of IngressRoutes that match those selectors.
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

// Get takes name of the IngressRoute, and returns the corresponding IngressRoute object, and an error if there is any.
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

// Create takes the representation of a IngressRoute and creates it.  Returns the server's representation of the IngressRoute, and an error, if there is any.
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

// Update takes the representation of a ingressRoute and updates it. Returns the server's representation of the ingressRoute, and an error, if there is any.
func (c *ingressRouteClient) Update(ctx context.Context, ir *IngressRoute, opts metav1.UpdateOptions) (*IngressRoute, error) {
	result := IngressRoute{}
	err := c.restClient.
		Put().
		Namespace(c.ns).
		Resource(IngressRouteResource).
		Name(ir.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ir).
		Do(ctx).
		Into(&result)
	return &result, err
}

// Watch returns a watch.Interface that watches the requested IngressRoute.
func (c *ingressRouteClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource(IngressRouteResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}

// Delete takes name of the IngressRoute and deletes it. Returns an error if one occurs.
func (c *ingressRouteClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.restClient.
		Delete().
		Namespace(c.ns).
		Resource(IngressRouteResource).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

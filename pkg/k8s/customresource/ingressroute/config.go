package ingressroute

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type Interface interface {
	IngressRoutes(namespace string) IngressRouteInterface
}

type CustomResourceClient struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*CustomResourceClient, error) {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, fmt.Errorf("create rest client failed: %w", err)
	}

	err = AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("register type definitions failed: %w", err)
	}

	return &CustomResourceClient{restClient: client}, nil
}

func (c *CustomResourceClient) IngressRoutes(namespace string) IngressRouteInterface {
	return &ingressRouteClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

package ingressroute

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Client manages communication with the Traefik IngressRoute.
type Client struct {
	clientset Interface
	logger    logging.Logger
}

// NewClient constructs a new Client.
func NewClient(clientset Interface, log logging.Logger) *Client {
	return &Client{
		clientset: clientset,
		logger:    log,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	Spec        IngressRouteSpec
}

// Set updates IngressRoute or creates it if it does not exist
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

	getObj, err := c.clientset.IngressRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			ing, err = c.clientset.IngressRoutes(namespace).Create(ctx, spec)
			if err != nil {
				return nil, fmt.Errorf("creating ingress route %s in namespace %s: %w", name, namespace, err)
			}
			return
		} else {
			return nil, fmt.Errorf("getting ingress route %s in namespace %s: %w", name, namespace, err)
		}
	}

	spec.ResourceVersion = getObj.GetResourceVersion()

	ing, err = c.clientset.IngressRoutes(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating ingress route %s in namespace %s: %w", name, namespace, err)
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

// GetNodes list Ingress Routes hosts using label as selector, for the given namespace. If label is empty, all Ingresses are listed.
func (c *Client) GetNodes(ctx context.Context, namespace, label string) (nodes []ingress.NodeInfo, err error) {
	c.logger.Debugf("listing IngressRoutes in namespace %s, label selector %s", namespace, label)
	ingressRoutes, err := c.clientset.IngressRoutes(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list ingress routes in namespace %s: %w", namespace, err)
	}

	if ingressRoutes != nil {
		c.logger.Debugf("found %d ingress routes in namespace %s", len(ingressRoutes.Items), namespace)

		for _, ingressRoute := range ingressRoutes.Items {
			for _, route := range ingressRoute.Spec.Routes {
				host := route.GetHost()
				if host != "" {
					nodes = append(nodes, ingress.NodeInfo{
						Name: ingressRoute.Name,
						Host: host,
					})
				}
			}
		}
	}

	return nodes, nil
}

package ingress

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/logging"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Ingress.
type Client struct {
	clientset kubernetes.Interface
	logger    logging.Logger
}

// NewClient constructs a new Client.
func NewClient(clientset kubernetes.Interface, log logging.Logger) *Client {
	return &Client{
		clientset: clientset,
		logger:    log,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	Spec        Spec
}

// NodeInfo
type NodeInfo struct {
	Name string
	Host string
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

	return ing, err
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

	return err
}

// GetNodes list Ingresses hosts using label as selector, for the given namespace. If label is empty, all Ingresses are listed.
func (c *Client) GetNodes(ctx context.Context, namespace, label string) (nodes []NodeInfo, err error) {
	c.logger.Debugf("listing Ingresses in namespace %s, label selector %s", namespace, label)
	ingreses, err := c.clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list ingresses in namespace %s: %w", namespace, err)
	}

	c.logger.Debugf("found %d ingresses in namespace %s", len(ingreses.Items), namespace)

	for _, ingress := range ingreses.Items {
		for _, rule := range ingress.Spec.Rules {
			if rule.Host != "" {
				nodes = append(nodes, NodeInfo{
					Name: ingress.Name,
					Host: rule.Host,
				})
				c.logger.Tracef("found Ingress %s in namespace %s with host %s", ingress.Name, namespace, rule.Host)
			}
		}
	}

	return nodes, nil
}

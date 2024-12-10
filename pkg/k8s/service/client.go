package service

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Service.
type Client struct {
	clientset       kubernetes.Interface
	inClusterDomain string
}

// NewClient constructs a new Client.
func NewClient(clientset kubernetes.Interface, inClusterDomain string) *Client {
	return &Client{
		clientset:       clientset,
		inClusterDomain: inClusterDomain,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	ServiceSpec Spec
}

// Set updates Service or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (svc *v1.Service, err error) {
	spec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: o.ServiceSpec.ToK8S(),
	}

	svc, err = c.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			svc, err = c.clientset.CoreV1().Services(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("creating service %s in namespace %s: %w", name, namespace, err)
			}
			return
		}
		return nil, fmt.Errorf("getting service %s in namespace %s: %w", name, namespace, err)
	}

	spec.ResourceVersion = svc.ResourceVersion
	spec.Spec.ClusterIP = svc.Spec.ClusterIP
	svc, err = c.clientset.CoreV1().Services(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating service %s in namespace %s: %w", name, namespace, err)
	}

	return
}

// Delete deletes Service
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting service %s in namespace %s: %w", name, namespace, err)
	}

	return
}

type NodeInfo struct {
	Name     string
	Endpoint string
}

// GetNodes returns list of nodes in the namespace with labelSelector. Nodes are filtered by api port.
// Endpoint is constructed from ClusterIP and api port.
func (c *Client) GetNodes(ctx context.Context, namespace, labelSelector string) (nodes []NodeInfo, err error) {
	svcs, err := c.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("listing services in namespace %s: %w", namespace, err)
	}

	// filter out services with api port and return clusterIP as Endpoint
	for _, svc := range svcs.Items {
		for _, port := range svc.Spec.Ports {
			if port.Name == "api" {
				nodes = append(nodes, NodeInfo{
					Name:     svc.Name,
					Endpoint: fmt.Sprintf("http://%s:%v", svc.Spec.ClusterIP, port.Port),
				})
			}
		}
	}

	return
}

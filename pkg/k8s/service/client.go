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
	ServiceSpec Spec
}

// Set updates Service or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: o.ServiceSpec.ToK8S(),
	}

	svc, err := c.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = c.clientset.CoreV1().Services(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("creating service %s in namespace %s: %v", name, namespace, err)
			}
			return
		}
		return fmt.Errorf("getting service %s in namespace %s: %v", name, namespace, err)
	}

	spec.ResourceVersion = svc.ResourceVersion
	spec.Spec.ClusterIP = svc.Spec.ClusterIP
	_, err = c.clientset.CoreV1().Services(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating service %s in namespacee %s: %v", name, namespace, err)
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
		return fmt.Errorf("deleting service %s in namespace %s: %v", name, namespace, err)
	}

	return
}

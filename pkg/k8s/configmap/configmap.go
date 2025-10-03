package configmap

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes ConfigMap.
type Client struct {
	clientset kubernetes.Interface
}

// NewClient constructs a new Client.
func NewClient(clientset kubernetes.Interface) *Client {
	return &Client{
		clientset: clientset,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	Immutable   bool
	Data        map[string]string
	BinaryData  map[string][]byte
}

// Set updates ConfigMap or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (cm *v1.ConfigMap, err error) {
	spec := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Immutable:  &o.Immutable,
		BinaryData: o.BinaryData,
		Data:       o.Data,
	}

	cm, err = c.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			cm, err = c.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("creating configmap %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return nil, fmt.Errorf("updating configmap %s in namespace %s: %w", name, namespace, err)
		}
	}

	return cm, err
}

// Delete deletes ConfigMap
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting configmap %s in namespace %s: %w", name, namespace, err)
	}

	return err
}

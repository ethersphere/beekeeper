package secret

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Secret.
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
	Immutable   bool
	Data        map[string][]byte
	StringData  map[string]string
	Type        string
}

// Set updates Secret of creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Immutable:  &o.Immutable,
		Data:       o.Data,
		StringData: o.StringData,
		Type:       v1.SecretType(o.Type),
	}

	_, err = c.clientset.CoreV1().Secrets(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = c.clientset.CoreV1().Secrets(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("creating secret %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return fmt.Errorf("updating secret %s in namespace %s: %w", name, namespace, err)
		}
	}

	return
}

// Delete deletes Secret
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting secret %s in namespace %s: %w", name, namespace, err)
	}

	return
}

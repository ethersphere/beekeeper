package namespace

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Namespace.
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
}

// Create creates namespace
func (c *Client) Create(ctx context.Context, name string) (*v1.Namespace, error) {
	spec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"created-by": fmt.Sprintf("beekeeper:%s", beekeeper.Version),
			},
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "beekeeper",
			},
		},
	}

	return c.clientset.CoreV1().Namespaces().Create(ctx, spec, metav1.CreateOptions{})
}

// Update updates namespace
func (c *Client) Update(ctx context.Context, name string, o Options) (*v1.Namespace, error) {
	spec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
	}

	return c.clientset.CoreV1().Namespaces().Update(ctx, spec, metav1.UpdateOptions{})
}

// Delete deletes namespace
func (c *Client) Delete(ctx context.Context, name string) (err error) {
	n, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	managedBy, ok := n.Labels["app.kubernetes.io/managed-by"]
	if !ok || managedBy != "beekeeper" {
		return fmt.Errorf("namespace %s is not managed by beekeeper, try kubectl", name)
	}

	err = c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("deleting namespace %s: %w", name, err)
	}

	return err
}

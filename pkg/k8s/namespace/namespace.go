package namespace

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Namespace.
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
}

// Create creates namespace
func (c *Client) Create(ctx context.Context, name string, o Options) (err error) {
	spec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
	}

	_, err = c.clientset.CoreV1().Namespaces().Create(ctx, spec, metav1.CreateOptions{})
	return
}

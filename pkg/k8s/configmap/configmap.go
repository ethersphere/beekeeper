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
	Data        map[string]string
	BinaryData  map[string][]byte
}

// Set creates ConfigMap, if ConfigMap already exists updates in place
func (c Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		BinaryData: o.BinaryData,
		Data:       o.Data,
	}

	_, err = c.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			fmt.Printf("configmap %s already exists in the namespace %s, updating the map\n", name, namespace)
			_, err = c.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}

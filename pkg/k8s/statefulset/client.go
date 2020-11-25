package statefulset

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes StatefulSet.
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
	Spec        StatefulSetSpec
}

// Set creates StatefulSet, if StatefulSet already exists updates in place
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: o.Spec.ToK8S(),
	}

	_, err = c.clientset.AppsV1().StatefulSets(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			fmt.Printf("statefulset %s already exists in the namespace %s, updating the statefulset\n", name, namespace)
			_, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}

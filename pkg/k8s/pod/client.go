package pod

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Pods.
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
	PodSpec     PodSpec
}

// Set updates Pod or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (pod *v1.Pod, err error) {
	spec := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: o.PodSpec.toK8S(),
	}

	pod, err = c.clientset.CoreV1().Pods(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			pod, err = c.clientset.CoreV1().Pods(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("creating pod %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return nil, fmt.Errorf("updating pod %s in namespace %s: %w", name, namespace, err)
		}
	}

	return
}

// Delete deletes Pod
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting pod %s in namespace %s: %w", name, namespace, err)
	}

	return
}

package statefulset

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/autoscaling/v1"
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

// Delete deletes StatefulSet
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting statefulset %s in namespace %s: %w", name, namespace, err)
	}

	return
}

// ReadyReplicas returns number of Pods created by the StatefulSet controller that have a Ready Condition
func (c *Client) ReadyReplicas(ctx context.Context, name, namespace string) (ready int32, err error) {
	s, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("getting ReadyReplicas from statefulset %s in namespace %s: %w", name, namespace, err)
	}
	ready = s.Status.ReadyReplicas

	return
}

// Scale scales StatefulSet
func (c *Client) Scale(ctx context.Context, name, namespace string, replicas int32) (err error) {
	scale := &v1.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ScaleSpec{
			Replicas: replicas,
		},
	}

	_, err = c.clientset.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("scaling statefulset %s in namespace %s: %w", name, namespace, err)
	}

	return
}

// Set updates StatefulSet or creates it if it does not exist
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

	_, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = c.clientset.AppsV1().StatefulSets(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("creating statefulset %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return fmt.Errorf("updating statefulset %s in namespace %s: %w", name, namespace, err)
		}
	}

	return
}

// StartedStatefulSets returns names of started StatefulSets
func (c *Client) StartedStatefulSets(ctx context.Context, namespace string) (started []string, err error) {
	statefulSets, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list statefulsets in namespace %s: %w", namespace, err)
	}

	for _, s := range statefulSets.Items {
		if s.Status.Replicas == 1 {
			started = append(started, s.Name)
		}
	}

	return
}

// StoppedStatefulSets returns names of stopped StatefulSets
func (c *Client) StoppedStatefulSets(ctx context.Context, namespace string) (stopped []string, err error) {
	statefulSets, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list statefulsets in namespace %s: %w", namespace, err)
	}

	for _, s := range statefulSets.Items {
		if s.Status.Replicas == 0 {
			stopped = append(stopped, s.Name)
		}
	}

	return
}

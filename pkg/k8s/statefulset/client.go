package statefulset

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes StatefulSet.
type Client struct {
	clientset kubernetes.Interface
	log       logging.Logger
}

// NewClient constructs a new Client.
func NewClient(clientset kubernetes.Interface, l logging.Logger) *Client {
	return &Client{
		clientset: clientset,
		log:       l,
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

// ReadyReplicasWatch returns number of Pods created by the StatefulSet controller that have a Ready Condition by watching events
func (c *Client) ReadyReplicasWatch(ctx context.Context, name, namespace string) (ready int32, err error) {
	watcher, err := c.clientset.AppsV1().StatefulSets(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return 0, fmt.Errorf("getting ready from statefulset %s in namespace %s: %w", name, namespace, err)
	}
	defer watcher.Stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Use a select statement to listen for either events from the watcher or a context cancellation
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
			c.log.Infof("%s is not ready yet", name)
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return 0, fmt.Errorf("watch channel closed")
			}
			// Extract the StatefulSet from the event
			statefulSet, ok := event.Object.(*appsv1.StatefulSet)
			if ok && statefulSet.Status.Replicas == statefulSet.Status.ReadyReplicas {
				ready = statefulSet.Status.ReadyReplicas
				return
			}
		}
	}
}

// RunningStatefulSets returns names of running StatefulSets
func (c *Client) RunningStatefulSets(ctx context.Context, namespace string) (running []string, err error) {
	statefulSets, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list statefulsets in namespace %s: %w", namespace, err)
	}

	for _, s := range statefulSets.Items {
		if s.Status.Replicas == 1 {
			running = append(running, s.Name)
		}
	}

	return
}

// Scale scales StatefulSet
func (c *Client) Scale(ctx context.Context, name, namespace string, replicas int32) (sc *v1.Scale, err error) {
	scale := &v1.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ScaleSpec{
			Replicas: replicas,
		},
	}

	sc, err = c.clientset.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("scaling statefulset %s in namespace %s: %w", name, namespace, err)
	}

	return
}

// Set updates StatefulSet or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (statefulSet *appsv1.StatefulSet, err error) {
	spec := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: o.Spec.ToK8S(),
	}

	statefulSet, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			statefulSet, err = c.clientset.AppsV1().StatefulSets(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("creating statefulset %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return nil, fmt.Errorf("updating statefulset %s in namespace %s: %w", name, namespace, err)
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

// UpdateImage updates StatefulSet image
func (c *Client) UpdateImage(ctx context.Context, name, namespace, image string) (err error) {
	statefulSet, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("getting statefulset %s in namespace %s: %w", name, namespace, err)
	}

	statefulSet.Spec.Template.Spec.Containers[0].Image = image

	_, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating statefulset %s in namespace %s: %w", name, namespace, err)
	}

	return nil
}

// GetUpdateStrategy returns the update strategy of the StatefulSet
func (c *Client) GetUpdateStrategy(ctx context.Context, name, namespace string) (strategy UpdateStrategy, err error) {
	statefulSet, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return UpdateStrategy{}, nil
		}
		return UpdateStrategy{}, fmt.Errorf("getting statefulset %s in namespace %s: %w", name, namespace, err)
	}

	return newUpdateStrategy(statefulSet.Spec.UpdateStrategy), nil
}

func (c *Client) UpdateCommand(ctx context.Context, namespace, name string, cmd []string) error {
	statefulSet, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting statefulset %s in namespace %s: %w", name, namespace, err)
	}

	// Update the command in the first container
	if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
		statefulSet.Spec.Template.Spec.Containers[0].Command = cmd
	} else {
		return fmt.Errorf("no containers found in statefulset %s in namespace %s", name, namespace)
	}

	_, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating command in statefulset %s in namespace %s: %w", name, namespace, err)
	}

	return nil
}

package pod

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Pods.
type Client struct {
	clientset kubernetes.Interface
	log       logging.Logger
}

// NewClient constructs a new Client.
func NewClient(clientset kubernetes.Interface, log logging.Logger) *Client {
	return &Client{
		clientset: clientset,
		log:       log,
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
func (c *Client) Delete(ctx context.Context, name, namespace string) (ok bool, err error) {
	if err = c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			c.log.Warningf("pod %s in namespace %s not found", name, namespace)
			return false, nil
		}
		return false, fmt.Errorf("deleting pod %s in namespace %s: %w", name, namespace, err)
	}

	return true, nil
}

func (c *Client) DeletePods(ctx context.Context, namespace, labelSelector string) (int, error) {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return 0, fmt.Errorf("listing pods in namespace %s: %w", namespace, err)
	}

	deletedCount := 0
	var deletionErrors []error

	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			if err := c.clientset.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
				c.log.Errorf("failed to delete pod %s in namespace %s: %v", pod.Name, namespace, err)
				deletionErrors = append(deletionErrors, err)
				continue
			}
			deletedCount++
		}
	}

	c.log.Debugf("attempted to delete %d pods; successfully deleted %d pods in namespace %s", len(pods.Items), deletedCount, namespace)

	if len(deletionErrors) > 0 {
		return deletedCount, fmt.Errorf("some pods failed to delete: %v", deletionErrors)
	}
	return deletedCount, nil
}

// WatchNewRunning detects new running Pods in the namespace and sends their IPs to the channel.
func (c *Client) WatchNewRunning(ctx context.Context, namespace, labelSelector string, newPodIps chan string) (err error) {
	c.log.Debugf("starting events watch in namespace %s, label selector %s", namespace, labelSelector)
	defer c.log.Infof("events watch done")
	defer close(newPodIps)

	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("getting pod events in namespace %s: %w", namespace, err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}
			switch event.Type {
			// case watch.Added: // already running pods
			case watch.Modified:
				pod, ok := event.Object.(*v1.Pod)
				if ok {
					if pod.Status.PodIP != "" && pod.ObjectMeta.DeletionTimestamp == nil && pod.Status.Phase == v1.PodRunning {
						newPodIps <- pod.Status.PodIP
					}
				}
			}
		}
	}
}

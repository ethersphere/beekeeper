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

// EventsWatch watches for events.
func (c *Client) EventsWatch(ctx context.Context, namespace string, operatorChan chan string) (err error) {
	c.log.Infof("starting events watch")
	defer c.log.Infof("events watch done")
	defer close(operatorChan)

	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		// TODO: add this label to beekeeper and filter on it => app.kubernetes.io/name=bee
		LabelSelector: "app.kubernetes.io/name=bee",
	})
	if err != nil {
		return fmt.Errorf("getting pod events in namespace %s: %w", namespace, err)
	}
	defer watcher.Stop()

	// listen for either events from the watcher or a context cancellation
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}
			switch event.Type {
			// case watch.Added: // if we want to do something with already running pods
			case watch.Modified:
				pod, ok := event.Object.(*v1.Pod)
				if ok {
					if pod.Status.PodIP != "" && pod.ObjectMeta.DeletionTimestamp == nil {
						c.log.Tracef("new pod event: {%s}, {%s}, {%s}, {%s}, {%v}", event.Type, pod.Name, pod.Status.Phase, pod.Status.PodIP, pod.ObjectMeta.DeletionTimestamp)
						operatorChan <- pod.Status.PodIP
					}
				}
			}
		}
	}
}

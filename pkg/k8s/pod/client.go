package pod

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
	appsv1 "k8s.io/api/apps/v1"
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

	return pod, nil
}

func (c *Client) Get(ctx context.Context, podName string, namespace string) (*v1.Pod, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting pod %s in namespace %s: %w", podName, namespace, err)
	}

	return pod, nil
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

// WatchNewRunning detects new running Pods in the namespace and sends them to the channel.
func (c *Client) WatchNewRunning(ctx context.Context, namespace, labelSelector string, newPods chan *v1.Pod) error {
	c.log.Debugf("starting events watch in namespace %s, label selector %s", namespace, labelSelector)
	defer c.log.Debug("events watch done")
	defer close(newPods)

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
						for _, condition := range pod.Status.Conditions {
							if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
								newPods <- pod
								break
							}
						}
					}
				}
			}
		}
	}
}

func (c *Client) GetControllingStatefulSet(ctx context.Context, name string, namespace string) (*appsv1.StatefulSet, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting pod %s in namespace %s: %w", name, namespace, err)
	}

	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil || controllerRef.Kind != "StatefulSet" {
		return nil, fmt.Errorf("pod %s in namespace %s is not controlled by a StatefulSet", name, namespace)
	}

	statefulSet, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, controllerRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting StatefulSet %s in namespace %s: %w", controllerRef.Name, namespace, err)
	}

	return statefulSet, nil
}

func (c *Client) WaitCompleted(ctx context.Context, pod *v1.Pod, namespace string) error {
	c.log.Infof("waiting for pod %s to complete", pod.Name)
	defer c.log.Debugf("watch for pod %s in namespace %s done", pod.Name, namespace)

	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", pod.Name),
	})
	if err != nil {
		return fmt.Errorf("getting watch for pod %s in namespace %s: %w", pod.Name, namespace, err)
	}
	defer watcher.Stop()

	watchCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	containerName := pod.Spec.Containers[0].Name
	podName := pod.Name

	var hasBeenRunning bool
	var hasBeenDeleted bool
	var hasBeenAdded bool

	c.log.Debugf("watching for DELETED -> ADDED -> RUNNING -> COMPLETED sequence on pod %s, container %s", podName, containerName)

	for {
		select {
		case event := <-watcher.ResultChan():
			c.log.Debugf("received event type %s for pod %s", event.Type, podName)

			if !hasBeenDeleted {
				if event.Type != watch.Deleted {
					continue
				}
				c.log.Debugf("pod %s has been deleted", podName)
				hasBeenDeleted = true
			}

			if !hasBeenAdded {
				if event.Type != watch.Added {
					continue
				}
				c.log.Debugf("pod %s has been added", podName)
				hasBeenAdded = true
			}

			if event.Type != watch.Modified {
				continue
			}

			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				c.log.Debugf("watch event is not a pod, skipping")
				continue
			}

			for _, status := range pod.Status.ContainerStatuses {
				if status.Name == containerName {
					// Step 1: Check if the container is RUNNING.
					if status.State.Running != nil {
						if !hasBeenRunning {
							c.log.Debugf("pod %s container %s is now RUNNING", podName, containerName)
							hasBeenRunning = true
						}
					}

					// Step 2: Check if the container is TERMINATED.
					if status.State.Terminated != nil && hasBeenRunning {
						termState := status.State.Terminated

						if termState.Reason == "Error" {
							return fmt.Errorf("pod %s container %s terminated with an error (ExitCode: %d)", podName, containerName, termState.ExitCode)
						}

						if termState.Reason == "Completed" {
							c.log.Infof("pod %s container %s completed successfully", podName, containerName)
							return nil
						}
					}
					break
				}
			}

		case <-watchCtx.Done():
			return fmt.Errorf("timed out waiting for pod %s to complete", podName)
		}
	}
}

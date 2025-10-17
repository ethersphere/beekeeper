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
	"k8s.io/apimachinery/pkg/util/wait"
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
		if pod.DeletionTimestamp == nil {
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
					if pod.Status.PodIP != "" && pod.DeletionTimestamp == nil && pod.Status.Phase == v1.PodRunning {
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

// PodRecreationState represents the different states in the pod recreation lifecycle
type PodRecreationState int

const (
	WaitingForDeletion PodRecreationState = iota
	WaitingForCreation
	WaitingForRunning
	WaitingForCompletion
	Completed
)

func (s PodRecreationState) String() string {
	switch s {
	case WaitingForDeletion:
		return "WaitingForDeletion"
	case WaitingForCreation:
		return "WaitingForCreation"
	case WaitingForRunning:
		return "WaitingForRunning"
	case WaitingForCompletion:
		return "WaitingForCompletion"
	case Completed:
		return "Completed"
	default:
		return "Unknown"
	}
}

// WaitForPodRecreationAndCompletion waits for a pod to go through the complete lifecycle:
// DELETED -> ADDED -> RUNNING -> COMPLETED
func (c *Client) WaitForPodRecreationAndCompletion(ctx context.Context, namespace, podName string) error {
	c.log.Debugf("waiting for pod %s to complete recreation and execution lifecycle", podName)
	defer c.log.Debugf("watch for pod %s in namespace %s done", podName, namespace)

	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", podName),
	})
	if err != nil {
		return fmt.Errorf("getting watch for pod %s in namespace %s: %w", podName, namespace, err)
	}
	defer watcher.Stop()

	watchCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Initialize state machine
	currentState := WaitingForDeletion
	c.log.Debugf("starting pod recreation lifecycle watch for %s, initial state: %s", podName, currentState)

	for {
		select {
		case event := <-watcher.ResultChan():
			newState, err := c.processEventInState(event, currentState, podName)
			if err != nil {
				return err
			}

			if newState != currentState {
				c.log.Debugf("pod %s transitioning from %s to %s", podName, currentState, newState)
				currentState = newState
			}

			if currentState == Completed {
				c.log.Debugf("pod %s container completed successfully", podName)
				return nil
			}

		case <-watchCtx.Done():
			return fmt.Errorf("timed out waiting for pod %s to complete (current state: %s)", podName, currentState)
		}
	}
}

// processEventInState handles state transitions based on the received event
func (c *Client) processEventInState(event watch.Event, currentState PodRecreationState, podName string) (PodRecreationState, error) {
	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		c.log.Debugf("watch event is not a pod, skipping")
		return currentState, nil
	}

	containerName := pod.Spec.Containers[0].Name

	switch currentState {
	case WaitingForDeletion:
		if event.Type == watch.Deleted {
			c.log.Debugf("pod %s has been deleted", podName)
			return WaitingForCreation, nil
		}
	case WaitingForCreation:
		if event.Type == watch.Added {
			c.log.Debugf("pod %s has been recreated", podName)
			return WaitingForRunning, nil
		}
	case WaitingForRunning:
		if event.Type == watch.Modified {
			for _, status := range pod.Status.ContainerStatuses {
				if status.Name == containerName && status.State.Running != nil {
					c.log.Debugf("pod %s container %s is now running", podName, containerName)
					return WaitingForCompletion, nil
				}
			}
		}
	case WaitingForCompletion:
		if event.Type == watch.Modified {
			for _, status := range pod.Status.ContainerStatuses {
				if status.Name == containerName && status.State.Terminated != nil {
					termState := status.State.Terminated
					if termState.Reason == "Error" {
						return currentState, fmt.Errorf("pod %s container %s terminated with an error (ExitCode: %d)", podName, containerName, termState.ExitCode)
					}
					if termState.Reason == "Completed" {
						return Completed, nil
					}
				}
			}
		}
	}
	return currentState, nil
}

// WaitForRunning polls a pod until its status is 'Running'.
// It does not wait for the containers within the pod to become ready.
func (c *Client) WaitForRunning(ctx context.Context, namespace, podName string) error {
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return wait.PollUntilContextCancel(pollCtx, 5*time.Second, true, func(context.Context) (bool, error) {
		pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("getting pod %s in namespace %s: %w", podName, namespace, err)
		}

		if pod.Status.Phase == v1.PodRunning {
			c.log.Debugf("pod %s has started and is in phase Running.", podName)
			return true, nil
		}

		if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodUnknown {
			return false, fmt.Errorf("pod %s entered a bad state: %s", podName, pod.Status.Phase)
		}

		c.log.Debugf("pod %s is in phase %s, waiting for Running...", podName, pod.Status.Phase)
		return false, nil
	})
}

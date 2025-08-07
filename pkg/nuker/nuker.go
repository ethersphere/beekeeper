package nuker

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry" // Add this import
)

type ClientConfig struct {
	Log                     logging.Logger
	K8sClient               *k8s.Client
	BeeClients              map[string]*bee.Client
	NeighborhoodArgProvider NeighborhoodArgProvider
}

type Client struct {
	log                     logging.Logger
	k8sClient               *k8s.Client
	beeClients              map[string]*bee.Client
	neighborhoodArgProvider NeighborhoodArgProvider
}

// originalState holds the original configuration of a StatefulSet before an update.
type originalState struct {
	cmd            []string
	updateStrategy v1.StatefulSetUpdateStrategy
	replicas       *int32
	readinessProbe *corev1.Probe
}

func New(cfg *ClientConfig) *Client {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	return &Client{
		log:                     cfg.Log,
		k8sClient:               cfg.K8sClient,
		beeClients:              cfg.BeeClients,
		neighborhoodArgProvider: cfg.NeighborhoodArgProvider,
	}
}

// Run sends a update command to the Bee clients in the Kubernetes cluster
func (c *Client) Run(ctx context.Context, namespace, labelSelector string, restartArgs []string) (err error) {
	c.log.Info("starting Bee cluster update")

	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}

	if len(restartArgs) == 0 {
		return errors.New("args cannot be empty")
	}

	// 1. Find all unique StatefulSets to update.
	statefulSetsMap, err := c.findStatefulSets(ctx, namespace, labelSelector)
	if err != nil {
		return fmt.Errorf("failed to find stateful sets: %w", err)
	}

	if len(statefulSetsMap) == 0 {
		return errors.New("no stateful sets found to update")
	}

	// 2. Iterate through each StatefulSet and apply the update and rollback procedure.
	c.log.Debugf("found %d stateful sets to update", len(statefulSetsMap))
	for name, ss := range statefulSetsMap {
		args, err := c.neighborhoodArgProvider.GetArgs(ctx, ss, restartArgs)
		if err != nil {
			return fmt.Errorf("failed to get neighborhood args for stateful set %s: %w", name, err)
		}

		c.log.Infof("updating stateful set %s", name)
		if err := c.updateAndRollbackStatefulSet(ctx, namespace, ss, args); err != nil {
			return fmt.Errorf("failed to update stateful set %s: %w", name, err)
		}
		c.log.Infof("successfully updated stateful set %s", name)
	}

	c.log.Info("Bee cluster update completed successfully")
	return nil
}

func (c *Client) findStatefulSets(ctx context.Context, namespace, labelSelector string) (map[string]*v1.StatefulSet, error) {
	statefulSetsMap := make(map[string]*v1.StatefulSet)

	if len(c.beeClients) > 0 {
		// If bee clients are provided, find their controlling StatefulSets.
		for _, client := range c.beeClients {
			// Skip if the client is already in the map.
			if _, ok := statefulSetsMap[client.Name()]; ok {
				continue
			}

			statefulSet, err := c.k8sClient.Pods.GetControllingStatefulSet(ctx, client.Name(), namespace)
			if err != nil {
				if k8serrors.IsNotFound(err) {
					// Fallback to getting the StatefulSet directly if the pod is not found.
					ss, getErr := c.k8sClient.StatefulSet.Get(ctx, client.Name(), namespace)
					if getErr != nil {
						return nil, fmt.Errorf("failed to get stateful set for Bee client %s: %w", client.Name(), getErr)
					}
					statefulSetsMap[ss.Name] = ss
					continue
				}
				return nil, fmt.Errorf("failed to get controlling stateful set for Bee client %s: %w", client.Name(), err)
			}
			statefulSetsMap[statefulSet.Name] = statefulSet
		}
	} else {
		// If no clients are provided, find StatefulSets using namespace and labels.
		statefulSets, err := c.k8sClient.StatefulSet.StatefulSets(ctx, namespace, labelSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to get stateful sets in namespace %s: %w", namespace, err)
		}
		for _, statefulSet := range statefulSets {
			statefulSetsMap[statefulSet.Name] = statefulSet
		}
	}

	return statefulSetsMap, nil
}

// updateAndRollbackStatefulSet orchestrates the full update cycle for a single StatefulSet.
func (c *Client) updateAndRollbackStatefulSet(ctx context.Context, namespace string, ss *v1.StatefulSet, restartArgs []string) error {
	// Work on a deep copy to avoid modifying the original object in the map.
	ss = ss.DeepCopy()
	updateArgs := []string{"bee", "db", "nuke", "--config=.bee.yaml", "--data-dir=/home/bee/.bee"}

	// 1. Save the original state of the first container and the StatefulSet spec.
	if len(ss.Spec.Template.Spec.Containers) == 0 {
		return errors.New("stateful set has no containers")
	}
	original := originalState{
		cmd:            restartArgs,
		updateStrategy: ss.Spec.UpdateStrategy,
		replicas:       ss.Spec.Replicas,
		readinessProbe: ss.Spec.Template.Spec.Containers[0].ReadinessProbe,
	}
	c.log.Debugf("saved original state for stateful set %s: updateStrategy=%v, replicas=%d", ss.Name, original.updateStrategy.Type, *original.replicas)

	// 2. Modify the StatefulSet for the update task using a retry loop.
	c.log.Debugf("updating stateful set %s with command: %v", ss.Name, updateArgs)

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of the StatefulSet before updating.
		latestSS, err := c.k8sClient.StatefulSet.Get(ctx, ss.Name, namespace)
		if err != nil {
			return fmt.Errorf("failed to get latest stateful set %s: %w", ss.Name, err)
		}

		// Apply the changes for the nuke task.
		latestSS.Spec.UpdateStrategy.Type = v1.OnDeleteStatefulSetStrategyType
		latestSS.Spec.UpdateStrategy.RollingUpdate = nil
		latestSS.Spec.Template.Spec.Containers[0].Command = updateArgs
		latestSS.Spec.Template.Spec.Containers[0].ReadinessProbe = nil

		return c.k8sClient.StatefulSet.Update(ctx, namespace, latestSS)
	}); err != nil {
		return fmt.Errorf("failed to apply update spec to stateful set %s: %w", ss.Name, err)
	}

	// 3. Sequentially delete each pod and wait for it to be recreated and complete the task.
	c.log.Infof("deleting pods in stateful set %s to trigger update task", ss.Name)
	if err := c.recreatePodsAndWait(ctx, namespace, ss, c.k8sClient.Pods.WaitForPodRecreationAndCompletion); err != nil {
		return fmt.Errorf("failed during pod update task for %s: %w", ss.Name, err)
	}
	c.log.Infof("all pods for %s completed the update task", ss.Name)

	// 4. Roll back the StatefulSet to its original configuration using a retry loop.
	c.log.Infof("rolling back stateful set %s to its original configuration", ss.Name)
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of the StatefulSet before updating.
		latestSS, err := c.k8sClient.StatefulSet.Get(ctx, ss.Name, namespace)
		if err != nil {
			return fmt.Errorf("failed to get latest stateful set %s: %w", ss.Name, err)
		}

		// Apply the original configuration for rollback.
		latestSS.Spec.UpdateStrategy = original.updateStrategy
		latestSS.Spec.Replicas = original.replicas
		latestSS.Spec.Template.Spec.Containers[0].Command = original.cmd
		latestSS.Spec.Template.Spec.Containers[0].ReadinessProbe = original.readinessProbe

		return c.k8sClient.StatefulSet.Update(ctx, namespace, latestSS)
	}); err != nil {
		return fmt.Errorf("failed to apply rollback spec to stateful set %s: %w", ss.Name, err)
	}

	// 5. Sequentially delete each pod again to trigger the rollback.
	c.log.Infof("deleting pods in stateful set %s to trigger rollback", ss.Name)
	if err := c.recreatePodsAndWait(ctx, namespace, ss, c.k8sClient.Pods.WaitForReady); err != nil {
		return fmt.Errorf("failed during pod rollback for %s: %w", ss.Name, err)
	}
	c.log.Infof("all pods for %s have been rolled back and are ready", ss.Name)

	return nil
}

// podWaitFunc is a function type that defines how to wait for a pod to reach a desired state.
type podWaitFunc func(ctx context.Context, namespace, podName string) error

// recreatePodsAndWait handles the process of deleting pods one by one and waiting for them
// to reach a specific state using the provided wait function.
func (c *Client) recreatePodsAndWait(ctx context.Context, namespace string, ss *v1.StatefulSet, waitFunc podWaitFunc) error {
	for _, podName := range getPodNames(ss) {
		c.log.Debugf("deleting pod %s", podName)

		// Delete the pod to trigger the StatefulSet controller to recreate it.
		if _, err := c.k8sClient.Pods.Delete(ctx, podName, namespace); err != nil {
			return fmt.Errorf("failed to delete pod %s: %w", podName, err)
		}

		c.log.Debugf("waiting for pod %s to be recreated and finish its process", podName)
		if err := waitFunc(ctx, namespace, podName); err != nil {
			return err
		}
		c.log.Infof("pod %s is ready", podName)
	}
	return nil
}

// getPodNames generates a list of pod names for a given StatefulSet based on its replicas.
// Each pod name represents a Bee node.
func getPodNames(ss *v1.StatefulSet) []string {
	if ss == nil || ss.Spec.Replicas == nil {
		return nil
	}

	replicas := int(*ss.Spec.Replicas)
	podNames := make([]string, replicas)
	for i := range replicas {
		podNames[i] = fmt.Sprintf("%s-%d", ss.Name, i)
	}
	return podNames
}

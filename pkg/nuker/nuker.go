package nuker

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/node"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
)

type ClientConfig struct {
	Log                   logging.Logger
	NodeProvider          node.NodeProvider
	K8sClient             *k8s.Client
	UseRandomNeighborhood bool
	Image                 string
}

type Client struct {
	log                   logging.Logger
	nodeProvider          node.NodeProvider
	k8sClient             *k8s.Client
	useRandomNeighborhood bool
	image                 string
}

func New(cfg *ClientConfig) *Client {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	return &Client{
		log:          cfg.Log,
		k8sClient:    cfg.K8sClient,
		nodeProvider: cfg.NodeProvider,
		image:        cfg.Image,
	}
}

// Run sends a update command to the Bee clients in the Kubernetes cluster
func (c *Client) Run(ctx context.Context, restartArgs []string) (err error) {
	c.log.Info("starting Bee cluster update")

	namespace := c.nodeProvider.Namespace()
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}

	if len(restartArgs) == 0 {
		return errors.New("args cannot be empty")
	}

	nodes, err := c.nodeProvider.GetNodes(ctx)
	if err != nil {
		return fmt.Errorf("node provider failed to get nodes: %w", err)
	}

	neighborhoodArgProvider := newNeighborhoodProvider(c.log, nodes, c.useRandomNeighborhood)

	// 1. Find all unique StatefulSets to update.
	statefulSetsMap, err := c.findStatefulSets(ctx, nodes, namespace)
	if err != nil {
		return fmt.Errorf("failed to find stateful sets: %w", err)
	}

	if len(statefulSetsMap) == 0 {
		return errors.New("no stateful sets found to update")
	}

	// 2. Iterate through each StatefulSet and apply the update and rollback procedure concurrently using errgroup.
	c.log.Infof("found %d stateful sets to update", len(statefulSetsMap))

	count := 0

	for name, ss := range statefulSetsMap {
		// Skip StatefulSets with 0 replicas
		if ss.Spec.Replicas == nil || *ss.Spec.Replicas == 0 {
			c.log.Infof("skipping stateful set %s: no replicas", name)
			continue
		}

		if neighborhoodArgProvider.UsesRandomNeighborhood() && *ss.Spec.Replicas != 1 {
			c.log.Warningf("stateful set %s has %d replicas, but random neighborhood is enabled; all pods will receive the same neighborhood value", name, *ss.Spec.Replicas)
		}

		podNames := getPodNames(ss)

		args, err := neighborhoodArgProvider.GetArgs(ctx, podNames[0], restartArgs)
		if err != nil {
			return fmt.Errorf("failed to get neighborhood args for stateful set %s: %w", name, err)
		}

		c.log.Infof("updating stateful set %s, with args: %v", name, args)
		if err := c.updateAndRollbackStatefulSet(ctx, namespace, ss, args); err != nil {
			return fmt.Errorf("failed to update stateful set %s: %w", name, err)
		}
		count++
		c.log.Infof("successfully updated stateful set %s", name)
	}

	c.log.Infof("nuked %d stateful sets", count)

	return nil
}

// NukeByStatefulSets sends a nuke command to the specified StatefulSets by name
func (c *Client) NukeByStatefulSets(ctx context.Context, namespace string, statefulSetNames []string, restartArgs []string) error {
	c.log.Info("starting Bee cluster nuke by StatefulSet names")

	if len(statefulSetNames) == 0 {
		return errors.New("stateful set names cannot be empty")
	}

	statefulSetsMap := make(map[string]*v1.StatefulSet)
	var orderedNames []string

	for _, name := range statefulSetNames {
		if _, exists := statefulSetsMap[name]; exists {
			continue
		}

		statefulSet, err := c.k8sClient.StatefulSet.Get(ctx, name, namespace)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				c.log.Warningf("stateful set %s not found, skipping", name)
				continue
			}
			return fmt.Errorf("failed to get stateful set %s: %w", name, err)
		}

		statefulSetsMap[name] = statefulSet
		orderedNames = append(orderedNames, name)
	}

	if len(orderedNames) == 0 {
		c.log.Warning("no valid stateful sets found to update")
		return nil
	}

	c.log.Infof("found %d stateful sets to update", len(orderedNames))

	count := 0

	// iterate in the same order as the input statefulSetNames
	for _, name := range orderedNames {
		ss := statefulSetsMap[name]

		if ss.Spec.Replicas == nil || *ss.Spec.Replicas == 0 {
			c.log.Warningf("skipping stateful set %s: no replicas", name)
			continue
		}

		c.log.Infof("updating stateful set %s, with args: %v", name, restartArgs)
		if err := c.updateAndRollbackStatefulSet(ctx, namespace, ss, restartArgs); err != nil {
			return fmt.Errorf("failed to update stateful set %s: %w", name, err)
		}
		count++
		c.log.Infof("successfully updated stateful set %s", name)
	}

	c.log.Infof("nuked %d stateful sets", count)

	return nil
}

func (c *Client) findStatefulSets(ctx context.Context, nodes node.NodeList, namespace string) (map[string]*v1.StatefulSet, error) {
	statefulSetsMap := make(map[string]*v1.StatefulSet)

	for _, node := range nodes {
		statefulSet, err := c.k8sClient.Pods.GetControllingStatefulSet(ctx, node.Name(), namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get controlling stateful set for node %s: %w", node.Name(), err)
		}
		statefulSetsMap[statefulSet.Name] = statefulSet
	}

	return statefulSetsMap, nil
}

// updateAndRollbackStatefulSet orchestrates the full update cycle for a single StatefulSet.
func (c *Client) updateAndRollbackStatefulSet(ctx context.Context, namespace string, ss *v1.StatefulSet, restartArgs []string) error {
	updateArgs := []string{"bee", "db", "nuke", "--config=.bee.yaml", "--data-dir=/home/bee/.bee"}

	// 1. Save the original state by creating a deep copy before any modifications.
	if len(ss.Spec.Template.Spec.Containers) == 0 {
		return errors.New("stateful set has no containers")
	}
	originalSS := ss.DeepCopy()

	c.log.Debugf("saved original state for stateful set %s: updateStrategy=%v, replicas=%d", ss.Name, originalSS.Spec.UpdateStrategy.Type, *originalSS.Spec.Replicas)

	// Ensure rollback happens regardless of success or failure
	defer func() {
		c.log.Debugf("rolling back stateful set %s to its original configuration", ss.Name)
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Fetch the latest version of the StatefulSet before updating.
			latestSS, err := c.k8sClient.StatefulSet.Get(ctx, ss.Name, namespace)
			if err != nil {
				return fmt.Errorf("failed to get latest stateful set %s: %w", ss.Name, err)
			}

			// Restore the original configuration from our deep copy
			latestSS.Spec.UpdateStrategy = originalSS.Spec.UpdateStrategy
			latestSS.Spec.Replicas = originalSS.Spec.Replicas
			latestSS.Spec.Template.Spec.Containers[0].Command = restartArgs
			latestSS.Spec.Template.Spec.Containers[0].ReadinessProbe = originalSS.Spec.Template.Spec.Containers[0].ReadinessProbe
			if c.image != "" {
				latestSS.Spec.Template.Spec.Containers[0].Image = c.image
			}

			return c.k8sClient.StatefulSet.Update(ctx, namespace, latestSS)
		}); err != nil {
			c.log.Errorf("failed to apply rollback spec to stateful set %s: %v", ss.Name, err)
			return
		}

		// Sequentially delete each pod again to trigger the rollback.
		c.log.Debugf("deleting pods in stateful set %s to trigger rollback", ss.Name)
		if err := c.recreatePodsAndWait(ctx, namespace, ss, c.k8sClient.Pods.WaitForRunning); err != nil {
			c.log.Errorf("failed during pod rollback for %s: %v", ss.Name, err)
			return
		}
		c.log.Debugf("all pods for %s have been rolled back and are ready", ss.Name)
	}()

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
	c.log.Debugf("deleting pods in stateful set %s to trigger update task", ss.Name)
	if err := c.recreatePodsAndWait(ctx, namespace, ss, c.k8sClient.Pods.WaitForPodRecreationAndCompletion); err != nil {
		return fmt.Errorf("failed during pod update task for %s: %w", ss.Name, err)
	}
	c.log.Debugf("all pods for %s completed the update task", ss.Name)

	return nil
}

// podWaitFunc is a function type that defines how to wait for a pod to reach a desired state.
type podWaitFunc func(ctx context.Context, namespace, podName string) error

// recreatePodsAndWait handles the process of deleting pods one by one and waiting for them
// to reach a specific state using the provided wait function.
func (c *Client) recreatePodsAndWait(ctx context.Context, namespace string, ss *v1.StatefulSet, waitFunc podWaitFunc) error {
	pods := getPodNames(ss)
	g, ctx := errgroup.WithContext(ctx)

	// This flow is designed for current Helm deployments, where Beekeeper uses only one replica per StatefulSet.
	// The nuke command is short-lived and causes pods to enter CrashLoopBackOff quickly after execution.
	// To reliably detect nuke execution, we start a watcher goroutine for each pod before deletion.
	// Pods are then deleted as quickly as possible to trigger immediate recreation, which is important due to the StatefulSet's pod management policy.
	// If the nuke command were long-running (e.g., running as a service with a readiness probe and status reporting), the flow could be adapted for more robust handling.
	// For faster pod recreation and deletion, consider setting podManagementPolicy: Parallel in the StatefulSet spec.
	// Note: Even with Parallel, StatefulSet rolling updates remain sequential; only pod creation/deletion is parallelized.

	// Start watcher goroutines for each pod
	for _, podName := range pods {
		g.Go(func() error {
			c.log.Debugf("waiting for pod %s to be recreated and finish its process", podName)
			if err := waitFunc(ctx, namespace, podName); err != nil {
				return fmt.Errorf("failed to wait for pod %s: %w", podName, err)
			}
			c.log.Debugf("pod %s is ready", podName)
			return nil
		})
	}

	// Start deleter goroutine
	g.Go(func() error {
		for _, podName := range pods {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			c.log.Debugf("deleting pod %s", podName)
			if _, err := c.k8sClient.Pods.Delete(ctx, podName, namespace); err != nil {
				return fmt.Errorf("failed to delete pod %s: %w", podName, err)
			}
		}
		return nil
	})

	return g.Wait()
}

// getPodNames generates a list of pod names for a given StatefulSet based on its replicas.
// Each pod name represents a Bee node.
func getPodNames(ss *v1.StatefulSet) []string {
	if ss == nil || ss.Spec.Replicas == nil {
		return nil
	}

	replicas := int(*ss.Spec.Replicas)
	if replicas <= 0 {
		return nil
	}

	podNames := make([]string, replicas)
	for i := range replicas {
		podNames[i] = fmt.Sprintf("%s-%d", ss.Name, i)
	}
	return podNames
}

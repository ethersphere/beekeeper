package check

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"golang.org/x/sync/errgroup"
)

// Check ...
type Check interface {
	Run(ctx context.Context, cluster *bee.Cluster, o Options) (err error)
}

// Options ...
type Options struct {
	MetricsEnabled bool
	MetricsPusher  *push.Pusher
}

// Stage ...
type Stage struct {
	Updates []Update
}

// Update represents details for updating a node group
type Update struct {
	NodeGroup string
	Actions   Actions
}

// Actions ...
type Actions struct {
	AddCount    int
	StartCount  int
	StopCount   int
	DeleteCount int
}

// Run runs check against the cluster
func Run(ctx context.Context, seed int64, cluster *bee.Cluster, check Check, options Options, stages []Stage) (err error) {
	fmt.Printf("root seed: %d\n", seed)

	if err := check.Run(ctx, cluster, options); err != nil {
		return err
	}

	for i, s := range stages {
		for _, u := range s.Updates {
			fmt.Printf("stage %d, node group %s, add %d, delete %d, start %d, stop %d\n", i, u.NodeGroup, u.Actions.AddCount, u.Actions.DeleteCount, u.Actions.StartCount, u.Actions.StopCount)

			rnd := random.PseudoGenerator(seed)
			ng := cluster.NodeGroup(u.NodeGroup)
			if err := updateNodeGroup(ctx, ng, u.Actions, i, rnd); err != nil {
				return err
			}
		}

		// wait at least 60s for deleted nodes to be removed from the peers list
		time.Sleep(65 * time.Second)
		if err := check.Run(ctx, cluster, options); err != nil {
			return err
		}
	}

	return
}

// RunConcurrently runs check against the cluster, cluster updates are executed concurrently
func RunConcurrently(ctx context.Context, cluster *bee.Cluster, check Check, options Options, stages []Stage, buffer int, seed int64, timeout time.Duration) (err error) {
	fmt.Printf("root seed: %d\n", seed)

	if err := check.Run(ctx, cluster, options); err != nil {
		return err
	}

	for i, s := range stages {
		for _, u := range s.Updates {
			fmt.Printf("stage %d, node group %s, add %d, delete %d, start %d, stop %d\n", i, u.NodeGroup, u.Actions.AddCount, u.Actions.DeleteCount, u.Actions.StartCount, u.Actions.StopCount)
			// make weighter buffer

			rnd := random.PseudoGenerator(seed)
			ng := cluster.NodeGroup(u.NodeGroup)
			if err := updateNodeGroupConcurrently(ctx, ng, u.Actions, i, buffer, rnd, timeout); err != nil {
				return err
			}
		}

		// wait at least 60s for deleted nodes to be removed from the peers list
		time.Sleep(65 * time.Second)
		if err := check.Run(ctx, cluster, options); err != nil {
			return err
		}
	}

	return
}

// updateNodeGroup updates node group by adding, deleting, starting and stopping it's nodes
func updateNodeGroup(ctx context.Context, ng *bee.NodeGroup, a Actions, stage int, rnd *rand.Rand) (err error) {
	// get info from the cluster
	running, err := ng.RunningNodes(ctx)
	if err != nil {
		return fmt.Errorf("running nodes: %w", err)
	}
	if len(running) < a.DeleteCount+a.StopCount {
		return fmt.Errorf("not enough running nodes for given parameters, running: %d, delete: %d, stop %d", len(running), a.DeleteCount, a.StopCount)
	}

	stopped, err := ng.StoppedNodes(ctx)
	if err != nil {
		return fmt.Errorf("stoped nodes: %w", err)
	}
	if len(stopped) < a.StartCount {
		return fmt.Errorf("not enough stopped nodes for given parameters, stopped: %d, start: %d", len(running), a.StartCount)
	}

	// plan execution
	var toAdd []string
	for i := 0; i < a.AddCount; i++ {
		toAdd = append(toAdd, fmt.Sprintf("bee-s%dn%d", stage, i))
	}
	toDelete, running := randomPick(rnd, running, a.DeleteCount)
	toStart, stopped := randomPick(rnd, stopped, a.StartCount)
	toStop, running := randomPick(rnd, running, a.StopCount)

	// add nodes
	for _, n := range toAdd {
		if err := ng.AddStartNode(ctx, n, bee.NodeOptions{}); err != nil {
			return fmt.Errorf("add start node %s: %w", n, err)
		}
		overlay, err := ng.NodeClient(n).Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", n, err)
		}
		fmt.Printf("node %s (%s) is added\n", n, overlay)
	}

	// delete nodes
	for _, n := range toDelete {
		overlay, err := ng.NodeClient(n).Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", n, err)
		}
		if err := ng.DeleteNode(ctx, n); err != nil {
			return fmt.Errorf("delete node %s: %w", n, err)
		}
		fmt.Printf("node %s (%s) is deleted\n", n, overlay)
	}

	// start nodes
	for _, n := range toStart {
		if err := ng.StartNode(ctx, n); err != nil {
			return fmt.Errorf("start node %s: %w", n, err)
		}
		overlay, err := ng.NodeClient(n).Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", n, err)
		}
		fmt.Printf("node %s (%s) is started\n", n, overlay)
	}

	// stop nodes
	for _, n := range toStop {
		overlay, err := ng.NodeClient(n).Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", n, err)
		}
		if err := ng.StopNode(ctx, n); err != nil {
			return fmt.Errorf("stop node %s: %w", n, err)
		}
		fmt.Printf("node %s (%s) is stopped\n", n, overlay)
	}

	return
}

// updateNodeGroupConcurrently updates node group concurrently
func updateNodeGroupConcurrently(ctx context.Context, ng *bee.NodeGroup, a Actions, stage, buff int, rnd *rand.Rand, timeout time.Duration) (err error) {
	// get info from the cluster
	running, err := ng.RunningNodes(ctx)
	if err != nil {
		return fmt.Errorf("running nodes: %w", err)
	}
	if len(running) < a.DeleteCount+a.StopCount {
		return fmt.Errorf("not enough running nodes for given parameters, running: %d, delete: %d, stop %d", len(running), a.DeleteCount, a.StopCount)
	}

	stopped, err := ng.StoppedNodes(ctx)
	if err != nil {
		return fmt.Errorf("stoped nodes: %w", err)
	}
	if len(stopped) < a.StartCount {
		return fmt.Errorf("not enough stopped nodes for given parameters, stopped: %d, start: %d", len(running), a.StartCount)
	}

	// plan execution
	var toAdd []string
	for i := 0; i < a.AddCount; i++ {
		toAdd = append(toAdd, fmt.Sprintf("bee-s%dn%d", stage, i))
	}
	toDelete, running := randomPick(rnd, running, a.DeleteCount)
	toStart, stopped := randomPick(rnd, stopped, a.StartCount)
	toStop, running := randomPick(rnd, running, a.StopCount)

	updateCtx, updateCancel := context.WithTimeout(ctx, timeout)
	defer updateCancel()
	updateGroup := new(errgroup.Group)
	updateSemaphore := make(chan struct{}, buff)

	// add nodes
	for _, n := range toAdd {
		n := n
		updateSemaphore <- struct{}{}
		updateGroup.Go(func() error {
			defer func() {
				<-updateSemaphore
			}()

			if err := ng.AddStartNode(updateCtx, n, bee.NodeOptions{}); err != nil {
				return fmt.Errorf("add start node %s: %w", n, err)
			}
			overlay, err := ng.NodeClient(n).Overlay(updateCtx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", n, err)
			}
			fmt.Printf("node %s (%s) is added\n", n, overlay)
			return nil
		})
	}

	// delete nodes
	for _, n := range toDelete {
		n := n
		updateSemaphore <- struct{}{}
		updateGroup.Go(func() error {
			defer func() {
				<-updateSemaphore
			}()

			overlay, err := ng.NodeClient(n).Overlay(updateCtx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", n, err)
			}
			if err := ng.DeleteNode(updateCtx, n); err != nil {
				return fmt.Errorf("delete node %s: %w", n, err)
			}
			fmt.Printf("node %s (%s) is deleted\n", n, overlay)
			return nil
		})
	}

	// start nodes
	for _, n := range toStart {
		n := n
		updateSemaphore <- struct{}{}
		updateGroup.Go(func() error {
			defer func() {
				<-updateSemaphore
			}()

			if err := ng.StartNode(updateCtx, n); err != nil {
				return fmt.Errorf("start node %s: %w", n, err)
			}
			overlay, err := ng.NodeClient(n).Overlay(updateCtx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", n, err)
			}
			fmt.Printf("node %s (%s) is started\n", n, overlay)
			return nil
		})
	}

	// stop nodes
	for _, n := range toStop {
		n := n
		updateSemaphore <- struct{}{}
		updateGroup.Go(func() error {
			defer func() {
				<-updateSemaphore
			}()

			overlay, err := ng.NodeClient(n).Overlay(updateCtx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", n, err)
			}
			if err := ng.StopNode(updateCtx, n); err != nil {
				return fmt.Errorf("stop node %s: %w", n, err)
			}
			fmt.Printf("node %s (%s) is stopped\n", n, overlay)
			return nil
		})
	}

	return updateGroup.Wait()
}

// randomPick randomly picks n elements from the list, and returns lists of picked and unpicked elements
func randomPick(rnd *rand.Rand, list []string, n int) (picked, unpicked []string) {
	for i := 0; i < n; i++ {
		index := rnd.Intn(len(list))
		picked = append(picked, list[index])
		list = append(list[:index], list[index+1:]...)
	}
	return picked, list
}

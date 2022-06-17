package beekeeper

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"golang.org/x/sync/errgroup"
)

// Action defines Beekeeper Action's interface. An action that
// needs to expose metrics should implement the metrics.Reporter
// interface.
type Action interface {
	Run(ctx context.Context, cluster orchestration.Cluster, o interface{}) (err error)
}

// Stage define stages for updating Bee
type Stage []Update

// Update represents details for updating a node group
type Update struct {
	NodeGroup string
	Actions   Actions
}

// Actions represents node group update actions
type Actions struct {
	AddCount    int
	StartCount  int
	StopCount   int
	DeleteCount int
}

// Run runs check against the cluster
func Run(ctx context.Context, cluster orchestration.Cluster, action Action, options interface{}, stages []Stage, seed int64, logger logging.Logger) (err error) {
	logger.Infof("root seed: %d\n", seed)

	if err := action.Run(ctx, cluster, options); err != nil {
		return err
	}

	for i, s := range stages {
		waitDeleted := false
		for _, u := range s {
			if u.Actions.DeleteCount > 0 {
				waitDeleted = true
			}

			logger.Infof("stage %d, node group %s, add %d, delete %d, start %d, stop %d\n", i, u.NodeGroup, u.Actions.AddCount, u.Actions.DeleteCount, u.Actions.StartCount, u.Actions.StopCount)

			rnd := random.PseudoGenerator(seed)
			ng, err := cluster.NodeGroup(u.NodeGroup)
			if err != nil {
				return err
			}
			if err := updateNodeGroup(ctx, ng, u.Actions, rnd, i, logger); err != nil {
				return err
			}
		}

		// wait at least 60s for deleted nodes to be removed from the peers list
		if waitDeleted {
			time.Sleep(60 * time.Second)
		}

		if err := action.Run(ctx, cluster, options); err != nil {
			return err
		}
	}

	return
}

// RunConcurrently runs check against the cluster, cluster updates are executed concurrently
func RunConcurrently(ctx context.Context, cluster orchestration.Cluster, action Action, options interface{}, stages []Stage, buffer int, seed int64, logger logging.Logger) (err error) {
	logger.Infof("root seed: %d\n", seed)

	if err := action.Run(ctx, cluster, options); err != nil {
		return err
	}

	for i, s := range stages {
		logger.Infof("starting stage %d\n", i)
		buffers := weightedBuffers(buffer, s)
		rnds := random.PseudoGenerators(seed, len(s))

		stageGroup := new(errgroup.Group)
		stageSemaphore := make(chan struct{}, buffer)

		waitDeleted := false
		for j, u := range s {
			j, u := j, u

			if u.Actions.DeleteCount > 0 {
				waitDeleted = true
			}

			stageSemaphore <- struct{}{}
			stageGroup.Go(func() error {
				defer func() {
					<-stageSemaphore
				}()

				logger.Infof("node group %s, add %d, delete %d, start %d, stop %d\n", u.NodeGroup, u.Actions.AddCount, u.Actions.DeleteCount, u.Actions.StartCount, u.Actions.StopCount)
				ng, err := cluster.NodeGroup(u.NodeGroup)
				if err != nil {
					return err
				}
				if err := updateNodeGroupConcurrently(ctx, ng, u.Actions, rnds[j], i, buffers[j], logger); err != nil {
					return err
				}

				logger.Infof("node group %s updated successfully\n", u.NodeGroup)
				return nil
			})
		}

		if err := stageGroup.Wait(); err != nil {
			return fmt.Errorf("stage %d failed: %w", i, err)
		}

		// wait 60s for deleted nodes to be removed from the peers list
		if waitDeleted {
			time.Sleep(60 * time.Second)
		}

		if err := action.Run(ctx, cluster, options); err != nil {
			return err
		}
	}

	return
}

// updateNodeGroup updates node group by adding, deleting, starting and stopping it's nodes
func updateNodeGroup(ctx context.Context, ng orchestration.NodeGroup, a Actions, rnd *rand.Rand, stage int, logger logging.Logger) (err error) {
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
		toAdd = append(toAdd, fmt.Sprintf("%s-s%dn%d", ng.Name(), stage, i))
	}
	toDelete, running := randomPick(rnd, running, a.DeleteCount)
	toStart, _ := randomPick(rnd, stopped, a.StartCount)
	toStop, _ := randomPick(rnd, running, a.StopCount)

	// add nodes
	for _, n := range toAdd {
		if err := ng.SetupNode(ctx, n, orchestration.NodeOptions{}, orchestration.FundingOptions{}); err != nil {
			return fmt.Errorf("add start node %s: %w", n, err)
		}
		c, err := ng.NodeClient(n)
		if err != nil {
			return err
		}
		overlay, err := c.Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", n, err)
		}
		logger.Infof("node %s (%s) is added\n", n, overlay)
	}

	// delete nodes
	for _, n := range toDelete {
		c, err := ng.NodeClient(n)
		if err != nil {
			return err
		}
		overlay, err := c.Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", n, err)
		}
		if err := ng.DeleteNode(ctx, n); err != nil {
			return fmt.Errorf("delete node %s: %w", n, err)
		}
		logger.Infof("node %s (%s) is deleted\n", n, overlay)
	}

	// start nodes
	for _, n := range toStart {
		if err := ng.StartNode(ctx, n); err != nil {
			return fmt.Errorf("start node %s: %w", n, err)
		}
		c, err := ng.NodeClient(n)
		if err != nil {
			return err
		}
		overlay, err := c.Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", n, err)
		}
		logger.Infof("node %s (%s) is started\n", n, overlay)
	}

	// stop nodes
	for _, n := range toStop {
		c, err := ng.NodeClient(n)
		if err != nil {
			return err
		}
		overlay, err := c.Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", n, err)
		}
		if err := ng.StopNode(ctx, n); err != nil {
			return fmt.Errorf("stop node %s: %w", n, err)
		}
		logger.Infof("node %s (%s) is stopped\n", n, overlay)
	}

	return
}

// updateNodeGroupConcurrently updates node group concurrently
func updateNodeGroupConcurrently(ctx context.Context, ng orchestration.NodeGroup, a Actions, rnd *rand.Rand, stage, buff int, logger logging.Logger) (err error) {
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
		toAdd = append(toAdd, fmt.Sprintf("%s-s%dn%d", ng.Name(), stage, i))
	}
	toDelete, running := randomPick(rnd, running, a.DeleteCount)
	toStart, _ := randomPick(rnd, stopped, a.StartCount)
	toStop, _ := randomPick(rnd, running, a.StopCount)

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

			if err := ng.SetupNode(ctx, n, orchestration.NodeOptions{}, orchestration.FundingOptions{}); err != nil {
				return fmt.Errorf("add start node %s: %w", n, err)
			}
			c, err := ng.NodeClient(n)
			if err != nil {
				return err
			}
			overlay, err := c.Overlay(ctx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", n, err)
			}
			logger.Infof("node %s (%s) is added\n", n, overlay)
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

			c, err := ng.NodeClient(n)
			if err != nil {
				return err
			}
			overlay, err := c.Overlay(ctx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", n, err)
			}
			if err := ng.DeleteNode(ctx, n); err != nil {
				return fmt.Errorf("delete node %s: %w", n, err)
			}
			logger.Infof("node %s (%s) is deleted\n", n, overlay)
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

			if err := ng.StartNode(ctx, n); err != nil {
				return fmt.Errorf("start node %s: %w", n, err)
			}
			c, err := ng.NodeClient(n)
			if err != nil {
				return err
			}
			overlay, err := c.Overlay(ctx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", n, err)
			}
			logger.Infof("node %s (%s) is started\n", n, overlay)
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

			c, err := ng.NodeClient(n)
			if err != nil {
				return err
			}
			overlay, err := c.Overlay(ctx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", n, err)
			}
			if err := ng.StopNode(ctx, n); err != nil {
				return fmt.Errorf("stop node %s: %w", n, err)
			}
			logger.Infof("node %s (%s) is stopped\n", n, overlay)
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

// weightedBuffers breaks buffer into smaller buffers for each update
func weightedBuffers(buffer int, s Stage) (buffers []int) {
	total := 0
	for _, u := range s {
		actions := u.Actions.AddCount + u.Actions.DeleteCount + u.Actions.StartCount + u.Actions.StopCount
		total += actions
	}

	for _, u := range s {
		actions := u.Actions.AddCount + u.Actions.DeleteCount + u.Actions.StartCount + u.Actions.StopCount
		buffers = append(buffers, buffer*actions/total)
	}

	return
}

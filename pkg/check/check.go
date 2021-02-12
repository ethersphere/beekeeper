package check

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options ...
type Options struct {
	MetricsEnabled bool
	MetricsPusher  *push.Pusher
}

// Check ...
type Check interface {
	Run(ctx context.Context, cluster *bee.Cluster, o Options) (err error)
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

	for i, stage := range stages {
		for j, update := range stage.Updates {
			fmt.Println("stage", i, "update", j, update.NodeGroup, update.Actions)

			rnd := random.PseudoGenerator(seed)
			ng := cluster.NodeGroup(update.NodeGroup)
			if err := updateNodeGroup(ctx, i, rnd, ng, update.Actions); err != nil {
				return err
			}
		}

		// run check here
		if err := check.Run(ctx, cluster, options); err != nil {
			return err
		}
	}

	return
}

func updateNodeGroup(ctx context.Context, i int, rnd *rand.Rand, ng *bee.NodeGroup, a Actions) (err error) {
	// delete nodes
	for j := 0; j < a.DeleteCount; j++ {
		nName := ng.NodesSorted()[rnd.Intn(ng.Size())]
		overlay, err := ng.NodeClient(nName).Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", nName, err)
		}
		if err := ng.DeleteNode(ctx, nName); err != nil {
			return fmt.Errorf("delete node %s: %w", nName, err)
		}
		fmt.Printf("node %s (%s) is deleted\n", nName, overlay)
	}

	// start nodes
	for j := 0; j < a.StartCount; j++ {
		stopped, err := ng.StoppedNodes(ctx)
		if err != nil {
			return fmt.Errorf("stoped nodes: %w", err)
		}
		if len(stopped) > 0 {
			nName := stopped[rnd.Intn(len(stopped))]
			if err := ng.StartNode(ctx, nName); err != nil {
				return fmt.Errorf("start node %s: %w", nName, err)
			}
			overlay, err := ng.NodeClient(nName).Overlay(ctx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", nName, err)
			}
			fmt.Printf("node %s (%s) is started\n", nName, overlay)
		}
	}

	// stop nodes
	for j := 0; j < a.StopCount; j++ {
		started, err := ng.StartedNodes(ctx)
		if err != nil {
			return fmt.Errorf("started nodes: %w", err)
		}
		if len(started) > 0 {
			nName := started[rnd.Intn(len(started))]
			overlay, err := ng.NodeClient(nName).Overlay(ctx)
			if err != nil {
				return fmt.Errorf("get node %s overlay: %w", nName, err)
			}
			if err := ng.StopNode(ctx, nName); err != nil {
				return fmt.Errorf("stop node %s: %w", nName, err)
			}
			fmt.Printf("node %s (%s) is stopped\n", nName, overlay)
		}
	}

	// add nodes
	for j := 0; j < a.AddCount; j++ {
		nName := fmt.Sprintf("bee-i%dn%d", i, j)
		if err := ng.AddStartNode(ctx, nName, bee.NodeOptions{}); err != nil {
			return fmt.Errorf("add start node %s: %w", nName, err)
		}
		overlay, err := ng.NodeClient(nName).Overlay(ctx)
		if err != nil {
			return fmt.Errorf("get node %s overlay: %w", nName, err)
		}
		fmt.Printf("node %s (%s) is added\n", nName, overlay)
	}
	return
}

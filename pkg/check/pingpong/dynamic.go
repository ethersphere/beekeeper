package pingpong

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Actions for dynamic behavior
type Actions struct {
	NodeGroup   string
	AddCount    int
	StartCount  int
	StopCount   int
	DeleteCount int
}

// CheckDynamic executes PingPong check on dynamic cluster
func CheckDynamic(ctx context.Context, cluster *bee.Cluster, o Options) (err error) {
	rnd := random.PseudoGenerator((o.Seed))
	fmt.Printf("Seed: %d\n", o.Seed)

	fmt.Println("Checking PingPong")
	if err := Check(ctx, cluster, o); err != nil {
		return fmt.Errorf("check pingpong: %w", err)
	}

	for i, a := range o.DynamicActions {
		ng := cluster.NodeGroup(a.NodeGroup)
		fmt.Printf("Start dynamic action on node group: %s\n", ng.Name())
		fmt.Printf("add %d nodes\n", a.AddCount)
		fmt.Printf("delete %d nodes\n", a.DeleteCount)
		fmt.Printf("start %d nodes\n", a.StartCount)
		fmt.Printf("stop %d nodes\n", a.StopCount)

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

		// wait at least 60s for deleted nodes to be removed from the peers list
		time.Sleep(65 * time.Second)

		fmt.Println("Checking PingPong")
		if err := Check(ctx, cluster, o); err != nil {
			return fmt.Errorf("check pingpong: %w", err)
		}
		fmt.Println("pingpong check completed successfully")
	}

	return nil
}

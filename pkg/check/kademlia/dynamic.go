package kademlia

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
	"github.com/ethersphere/beekeeper/pkg/random"
)

func checkKademliaD(topologies bee.ClusterTopologies) (err error) {
	for _, v := range topologies {
		for n, t := range v {
			if t.Depth == 0 {
				fmt.Printf("Node %s. Kademlia not healthy. Depth %d. Node: %s\n", n, t.Depth, t.Overlay)
				fmt.Printf("Error: %v\n", errKadmeliaNotHealthy.Error())
			}

			fmt.Printf("Node %s. Population: %d. Connected: %d. Depth: %d. Node: %s\n", n, t.Population, t.Connected, t.Depth, t.Overlay)
			for k, b := range t.Bins {
				binDepth, err := strconv.Atoi(strings.Split(k, "_")[1])
				if err != nil {
					fmt.Printf("Error: node %s: %v\n", n, err)
				}
				fmt.Printf("Bin %d. Population: %d. Connected: %d.\n", binDepth, b.Population, b.Connected)
				if binDepth < t.Depth && b.Connected < 1 {
					fmt.Printf("Error: %v\n", errKadmeliaBinConnected.Error())
				}

				if binDepth >= t.Depth && len(b.DisconnectedPeers) > 0 {
					fmt.Printf("Error: %v, %s\n", errKadmeliaBinDisconnected.Error(), b.DisconnectedPeers)
				}
			}
		}
	}

	return
}

// Actions for dynamic behavior
type Actions struct {
	NodeGroup   string
	AddCount    int
	StartCount  int
	StopCount   int
	DeleteCount int
}

// CheckDynamic executes Kademlia topology check on dynamic cluster
func CheckDynamic(ctx context.Context, cluster *bee.Cluster, o Options) (err error) {
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	fmt.Println("Checking connectivity")
	err = fullconnectivity.Check(ctx, cluster)
	if err != nil {
		fmt.Printf("Full connectivity not present: %v\n", err)
	} else {
		fmt.Printf("Full connectivity present\n")
	}

	topologies, err := cluster.Topologies(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Checking Kademlia")
	if err := checkKademliaD(topologies); err != nil {
		return fmt.Errorf("check Kademlia: %w", err)
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
			if err := ng.DeleteNode(ctx, nName); err != nil {
				return fmt.Errorf("delete node %s: %w", nName, err)
			}
			fmt.Printf("node %s is deleted\n", nName)
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
				fmt.Printf("node %s is started\n", nName)
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
				if err := ng.StopNode(ctx, nName); err != nil {
					return fmt.Errorf("stop node %s: %w", nName, err)
				}
				fmt.Printf("node %s is stopped\n", nName)
			}
		}

		// add nodes
		for j := 0; j < a.AddCount; j++ {
			nName := fmt.Sprintf("bee-i%dn%d", i, j)
			if err := ng.AddStartNode(ctx, nName, bee.NodeOptions{}); err != nil {
				return fmt.Errorf("add start node %s: %w", nName, err)
			}
			fmt.Printf("node %s is added\n", nName)
		}

		time.Sleep(5 * time.Second)

		fmt.Println("Checking connectivity")
		err = fullconnectivity.Check(ctx, cluster)
		if err != nil {
			fmt.Printf("Full connectivity not present\n")
		} else {
			fmt.Printf("Full connectivity present\n")
		}

		topologies, err := cluster.Topologies(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Checking Kademlia")
		if err := checkKademliaD(topologies); err != nil {
			fmt.Printf("Kademlia check failed: %v\n", err)
		}
	}

	return nil
}

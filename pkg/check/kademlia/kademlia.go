package kademlia

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
	"github.com/ethersphere/beekeeper/pkg/random"
)

var (
	errKadmeliaNotHealthy      = errors.New("kademlia not healthy")
	errKadmeliaBinConnected    = errors.New("at least 1 connected peer is required in a bin which is shallower than depth")
	errKadmeliaBinDisconnected = errors.New("peers disconnected at proximity order >= depth. Peers: %s")
)

// Options represents kademlia check options
type Options struct {
	Seed           int64
	DynamicActions []Actions
}

// Check executes Kademlia topology check on cluster
func Check(ctx context.Context, cluster *bee.Cluster) (err error) {
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
	if err := checkKademlia(topologies); err != nil {
		return fmt.Errorf("Kademlia check: %v", err)
	}

	return
}

// Actions ...
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
		fmt.Printf("Full connectivity not present\n")
	} else {
		fmt.Printf("Full connectivity present\n")
	}

	topologies, err := cluster.Topologies(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Checking Kademlia")
	if err := checkKademlia(topologies); err != nil {
		return fmt.Errorf("Kademlia check: %v", err)
	}

	for i, a := range o.DynamicActions {
		ng := cluster.NodeGroup(a.NodeGroup)
		fmt.Println(ng.Name(), a.AddCount, a.StartCount, a.StopCount, a.DeleteCount)

		// delete nodes
		for j := 0; j < a.DeleteCount; j++ {
			nName := ng.NodesSorted()[rnd.Intn(ng.Size())]
			if err := ng.DeleteNode(ctx, nName); err != nil {
				return fmt.Errorf("dynamic kademlia iteration %d round %d: %v", i, j, err)
			}
			fmt.Printf("node %s is deleted\n", nName)
		}

		// start nodes
		for j := 0; j < a.StartCount; j++ {
			stopped, err := ng.StoppedNodes(ctx)
			if err != nil {
				return fmt.Errorf("dynamic kademlia iteration %d round %d: %v", i, j, err)
			}
			if len(stopped) > 0 {
				nName := stopped[rnd.Intn(len(stopped))]
				if err := ng.StartNode(ctx, nName); err != nil {
					return fmt.Errorf("dynamic kademlia iteration %d round %d: %v", i, j, err)
				}
				fmt.Printf("node %s is started\n", nName)
			}
		}

		// stop nodes
		for j := 0; j < a.StopCount; j++ {
			started, err := ng.StartedNodes(ctx)
			if err != nil {
				return fmt.Errorf("dynamic kademlia iteration %d round %d: %v", i, j, err)
			}
			if len(started) > 0 {
				nName := started[rnd.Intn(len(started))]
				if err := ng.StopNode(ctx, nName); err != nil {
					return fmt.Errorf("dynamic kademlia iteration %d round %d: %v", i, j, err)
				}
				fmt.Printf("node %s is stopped\n", nName)
			}
		}

		// add nodes
		for j := 0; j < a.AddCount; j++ {
			nName := fmt.Sprintf("bee-i%dn%d", i, j)
			if err := ng.AddStartNode(ctx, nName, bee.NodeOptions{}); err != nil {
				return fmt.Errorf("dynamic kademlia iteration %d round %d: %v", i, j, err)
			}
			fmt.Printf("node %s is added\n", nName)
		}

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
		if err := checkKademlia(topologies); err != nil {
			return fmt.Errorf("Kademlia check: %v", err)
		}
	}

	return
}

func checkKademlia(topologies bee.ClusterTopologies) (err error) {
	for _, v := range topologies {
		for n, t := range v {
			if t.Depth == 0 {
				fmt.Printf("Node %s. Kademlia not healthy. Depth %d. Node: %s\n", n, t.Depth, t.Overlay)
				return errKadmeliaNotHealthy
			}

			fmt.Printf("Node %s. Population: %d. Connected: %d. Depth: %d. Node: %s\n", n, t.Population, t.Connected, t.Depth, t.Overlay)
			for k, b := range t.Bins {
				binDepth, err := strconv.Atoi(strings.Split(k, "_")[1])
				if err != nil {
					return fmt.Errorf("node %s: %w", n, err)
				}
				fmt.Printf("Bin %d. Population: %d. Connected: %d.\n", binDepth, b.Population, b.Connected)
				if binDepth < t.Depth && b.Connected < 1 {
					return errKadmeliaBinConnected
				}

				if binDepth >= t.Depth && len(b.DisconnectedPeers) > 0 {
					return fmt.Errorf(errKadmeliaBinDisconnected.Error(), b.DisconnectedPeers)
				}
			}
		}
	}

	return
}

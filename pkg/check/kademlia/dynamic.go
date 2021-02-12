package kademlia

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
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

// CheckDynamic executes Kademlia topology check on dynamic cluster
func CheckDynamic(ctx context.Context, cluster *bee.Cluster, o Options) (err error) {
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

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
			running, err := ng.RunningNodes(ctx)
			if err != nil {
				return fmt.Errorf("running nodes: %w", err)
			}
			if len(running) > 0 {
				nName := running[rnd.Intn(len(running))]
				overlay, err := ng.NodeClient(nName).Overlay(ctx)
				if err != nil {
					return fmt.Errorf("get node %s overlay: %w", nName, err)
				}
				if err := ng.DeleteNode(ctx, nName); err != nil {
					return fmt.Errorf("delete node %s: %w", nName, err)
				}
				fmt.Printf("node %s (%s) is deleted\n", nName, overlay)
			}
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
				fmt.Printf("node %s is started\n", nName)
				fmt.Printf("node %s (%s) is started\n", nName, overlay)
			}
		}

		// stop nodes
		for j := 0; j < a.StopCount; j++ {
			running, err := ng.RunningNodes(ctx)
			if err != nil {
				return fmt.Errorf("running nodes: %w", err)
			}
			if len(running) > 0 {
				nName := running[rnd.Intn(len(running))]
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

		time.Sleep(5 * time.Second)

		topologies, err := cluster.Topologies(ctx)
		if err != nil {
			return err
		}

		fmt.Println("kademlia check running")
		if err := checkKademliaD(topologies); err != nil {
			return err
		}
		fmt.Println("kademlia check completed successfully")
	}

	return nil
}

// checkKademliaD checks that for each topology, each node is connected to all
// peers that are within depth and that are online. Online-ness is assumed by the list
// of topologies (i.e. if we have the peer's topology, it is assumed it is online).
func checkKademliaD(topologies bee.ClusterTopologies) error {
	overlays := allOverlays(topologies)
	culprits := make(map[string][]swarm.Address)
	for _, nodeGroup := range topologies {
		for k, t := range nodeGroup {
			if t.Depth == 0 {
				return fmt.Errorf("node %s, address %s: %w", k, t.Overlay, errKadmeliaNotHealthy)
			}

			expNodes := nodesInDepth(uint8(t.Depth), t.Overlay, overlays)
			var nodes []swarm.Address

			fmt.Printf("Node %s. Population: %d. Connected: %d. Depth: %d. Node: %s. Expecting %d nodes within depth.\n", k, t.Population, t.Connected, t.Depth, t.Overlay, len(expNodes))

			for k, b := range t.Bins {
				bin, err := strconv.Atoi(strings.Split(k, "_")[1])
				if err != nil {
					fmt.Printf("Error: node %s: %v\n", k, err)
				}

				if bin >= t.Depth {
					nodes = append(nodes, b.ConnectedPeers...)
				}
			}

			if c := verifyNodes(expNodes, nodes); len(c) > 0 {
				culprits[t.Overlay.String()] = c
			}
		}
	}

	if len(culprits) > 0 {
		errmsg := ""
		for node, c := range culprits {
			msg := fmt.Sprintf("node %s expected connection to:\n", node)
			for _, addr := range c {
				msg += addr.String() + "\n"
			}

			errmsg += msg
		}
		return errors.New(errmsg)
	}

	return nil
}

func allOverlays(t bee.ClusterTopologies) []swarm.Address {
	var addrs []swarm.Address
	for _, nodeGroup := range t {
		for _, t := range nodeGroup {
			addrs = append(addrs, t.Overlay)
		}
	}
	return addrs
}

func nodesInDepth(d uint8, pivot swarm.Address, addrs []swarm.Address) []swarm.Address {
	var addrsInDepth []swarm.Address
	for _, addr := range addrs {
		if addr.Equal(pivot) {
			continue
		}
		if swarm.Proximity(pivot.Bytes(), addr.Bytes()) >= d {
			addrsInDepth = append(addrsInDepth, addr)
		}
	}
	return addrsInDepth
}

// verifyNodes verifies that all addresses in exp exist in nodes.
// returns a list of missing connections if any exist.
func verifyNodes(exp, nodes []swarm.Address) []swarm.Address {
	var culprits []swarm.Address

OUTER:
	for _, e := range exp {
		for _, n := range nodes {
			if e.Equal(n) {
				continue OUTER
			}
		}
		culprits = append(culprits, e)
	}

	return culprits
}

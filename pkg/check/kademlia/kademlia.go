package kademlia

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
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
		fmt.Printf("Full connectivity not present: %v\n", err)
	} else {
		fmt.Printf("Full connectivity present\n")
	}

	topologies, err := cluster.Topologies(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Checking Kademlia")
	if err := checkKademlia(topologies); err != nil {
		return fmt.Errorf("check Kademlia: %w", err)
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

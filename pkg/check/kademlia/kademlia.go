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
	errKademliaFullConnectivity = errors.New("full connectivity present")
	errKadmeliaNotHealthy       = errors.New("kademlia not healthy")
	errKadmeliaBinConnected     = errors.New("at least 1 connected peer is required in a bin which is shallower than depth")
	errKadmeliaBinDisconnected  = errors.New("peers disconnected at proximity order >= depth. Peers: %s")
)

// Check executes Kademlia topology check on cluster
func Check(cluster *bee.DynamicCluster) (err error) {
	ctx := context.Background()

	fmt.Printf("Checking for full connectivity:\n")
	if err := fullconnectivity.Check(cluster); err == nil {
		return errKademliaFullConnectivity
	}
	fmt.Printf("Full connectivity not present, continuing with kademlia topology check\n")

	topologies, err := cluster.Topologies(ctx)
	if err != nil {
		return err
	}

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

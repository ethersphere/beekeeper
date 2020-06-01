package kademlia

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

var (
	errKademliaFullConnectivity = errors.New("full connectivity present")
	errKadmeliaNotHealthy       = errors.New("kademlia not healthy")
	errKadmeliaBinConnected     = errors.New("at least 2 connected peers are required in a bin which is shallower than depth")
	errKadmeliaBinDisconnected  = errors.New("peers disconnected at proximity order >= depth. Peers: %s")
)

// Check executes Kademlia topology check on cluster
func Check(cluster bee.Cluster) (err error) {
	ctx := context.Background()

	if err := fullconnectivity.Check(cluster); err == nil {
		return errKademliaFullConnectivity
	}

	topologies, err := cluster.Topologies(ctx)
	if err != nil {
		return err
	}

	for i, t := range topologies {
		if t.Depth == 0 {
			fmt.Printf("Node %d. Kademlia not healthy. Depth %d. Node: %s\n", i, t.Depth, t.Overlay)
			return errKadmeliaNotHealthy
		}

		fmt.Printf("Node %d. Population: %d. Connected: %d. Depth: %d. Node: %s\n", i, t.Population, t.Connected, t.Depth, t.Overlay)
		for k, b := range t.Bins {
			binDepth, err := strconv.Atoi(strings.Split(k, "_")[1])
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			fmt.Printf("Bin %d. Population: %d. Connected: %d.\n", binDepth, b.Population, b.Connected)
			if binDepth < t.Depth && b.Connected < 2 {
				return errKadmeliaBinConnected
			}

			if binDepth >= t.Depth && len(b.DisconnectedPeers) > 0 {
				return fmt.Errorf(errKadmeliaBinDisconnected.Error(), b.DisconnectedPeers)
			}
		}
	}

	return
}

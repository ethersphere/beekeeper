package kademlia

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// Options represents check options
type Options struct {
	Dynamic bool
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		Dynamic: false,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	topologies, err := cluster.Topologies(ctx)
	if err != nil {
		return err
	}

	if o.Dynamic {
		return checkKademliaD(topologies, c.logger)
	}

	return checkKademlia(topologies, c.logger)
}

var (
	errKadmeliaNotHealthy      = errors.New("kademlia not healthy")
	errKadmeliaBinConnected    = errors.New("at least 1 connected peer is required in a bin which is shallower than depth")
	errKadmeliaBinDisconnected = errors.New("peers disconnected at proximity order >= depth. Peers: %s")
)

func checkKademlia(topologies orchestration.ClusterTopologies, logger logging.Logger) error {
	for _, v := range topologies {
		for n, t := range v {
			if t.Depth == 0 {
				logger.Infof("Node %s. Kademlia not healthy. Depth %d. Node: %s", n, t.Depth, t.Overlay)
				return errKadmeliaNotHealthy
			}

			logger.Infof("Node %s. Population: %d. Connected: %d. Depth: %d. Node: %s", n, t.Population, t.Connected, t.Depth, t.Overlay)
			for k, b := range t.Bins {
				binDepth, err := strconv.Atoi(strings.Split(k, "_")[1])
				if err != nil {
					return fmt.Errorf("node %s: %w", n, err)
				}
				logger.Infof("Bin %d. Population: %d. Connected: %d.", binDepth, b.Population, b.Connected)
				if binDepth < t.Depth && b.Connected < 1 {
					return errKadmeliaBinConnected
				}

				if binDepth >= t.Depth && len(b.DisconnectedPeers) > 0 {
					return fmt.Errorf(errKadmeliaBinDisconnected.Error(), b.DisconnectedPeers)
				}
			}
		}
	}

	return nil
}

// checkKademliaD checks that for each topology, each node is connected to all
// peers that are within depth and that are online. Online-ness is assumed by the list
// of topologies (i.e. if we have the peer's topology, it is assumed it is online).
func checkKademliaD(topologies orchestration.ClusterTopologies, logger logging.Logger) error {
	overlays := allOverlays(topologies)
	culprits := make(map[string][]swarm.Address)
	for _, nodeGroup := range topologies {
		for k, t := range nodeGroup {
			if t.Depth == 0 {
				return fmt.Errorf("node %s, address %s: %w", k, t.Overlay, errKadmeliaNotHealthy)
			}

			expNodes := nodesInDepth(uint8(t.Depth), t.Overlay, overlays)
			var nodes []swarm.Address

			logger.Infof("Node %s. Population: %d. Connected: %d. Depth: %d. Node: %s. Expecting %d nodes within depth.", k, t.Population, t.Connected, t.Depth, t.Overlay, len(expNodes))

			for k, b := range t.Bins {
				bin, err := strconv.Atoi(strings.Split(k, "_")[1])
				if err != nil {
					logger.Infof("Error: node %s: %v", k, err)
				}

				if bin >= t.Depth {
					for _, nn := range b.ConnectedPeers {
						nodes = append(nodes, nn.Address)
					}
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

func allOverlays(t orchestration.ClusterTopologies) []swarm.Address {
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

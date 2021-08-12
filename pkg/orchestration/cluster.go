package orchestration

import (
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
)

// ClusterAddresses represents addresses of all nodes in the cluster
type ClusterAddresses map[string]NodeGroupAddresses

// ClusterBalances represents balances of all nodes in the cluster
type ClusterBalances map[string]NodeGroupBalances

// ClusterOverlays represents overlay addresses of all nodes in the cluster
type ClusterOverlays map[string]NodeGroupOverlays

// RandomOverlay returns a random overlay from a random NodeGroup
func (c ClusterOverlays) Random(r *rand.Rand) (nodeGroup string, nodeName string, overlay swarm.Address) {
	i := r.Intn(len(c))
	var (
		ng, name string
		ngo      NodeGroupOverlays
		o        swarm.Address
	)
	for n, v := range c {
		if i == 0 {
			ng = n
			ngo = v
			break
		}
		i--
	}

	i = r.Intn(len(ngo))

	for n, v := range ngo {
		if i == 0 {
			name = n
			o = v
			break
		}
		i--
	}
	return ng, name, o
}

// ClusterPeers represents peers of all nodes in the cluster
type ClusterPeers map[string]NodeGroupPeers

// ClusterSettlements represents settlements of all nodes in the cluster
type ClusterSettlements map[string]NodeGroupSettlements

// ClusterTopologies represents Kademlia topology of all nodes in the cluster
type ClusterTopologies map[string]NodeGroupTopologies

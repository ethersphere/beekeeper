package check

import (
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// PeerCount ...
func PeerCount(o Options) (err error) {
	var nodes []bee.Node
	for i := 0; i < o.NodeCount; i++ {
		node, err := bee.NewNode(fmt.Sprintf(o.NodeURLTemplate, i))
		if err != nil {
			return err
		}

		nodes = append(nodes, *node)
	}

	if err := testPeerCount(nodes, o.NodeCount); err != nil {
		return err
	}

	return
}

var errPeerCount = errors.New("peer count")

func testPeerCount(nodes []bee.Node, peerCount int) (err error) {
	for i, n := range nodes {
		if len(n.Peers.Peers) == peerCount {
			fmt.Println(fmt.Sprintf("Node %d passed.", i))
		} else {
			fmt.Println(fmt.Sprintf("Node %d failed. Peers: %d/%d", i, len(n.Peers.Peers), peerCount))
			return errPeerCount
		}
	}

	return nil
}

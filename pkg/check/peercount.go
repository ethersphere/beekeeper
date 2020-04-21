package check

import (
	"errors"
	"fmt"
	"log"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options ...
type Options struct {
	NodeCount       int
	NodeURLTemplate string
}

// PeerCount ...
func PeerCount(o Options) {
	var nodes []bee.Node
	for i := 0; i < o.NodeCount; i++ {
		node, err := bee.NewNode(fmt.Sprintf(o.NodeURLTemplate, i))
		if err != nil {
			log.Fatalln(err)
		}

		nodes = append(nodes, *node)

		fmt.Printf("Node %d:\n", i)
		printNode(*node)
	}
}

func printNode(node bee.Node) {
	fmt.Println(fmt.Sprintf("Overlay: %v", node.Addresses.Overlay))
	for _, u := range node.Addresses.Underlay {
		fmt.Println(fmt.Sprintf("Underlay: %v", u))
	}
	for _, p := range node.Peers.Peers {
		fmt.Println(fmt.Sprintf("Peer: %v", p.Address))
	}
}

var errPeerCount = errors.New("peer count")

func testPeerCount(nodes []bee.Node, peerCount int) error {
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

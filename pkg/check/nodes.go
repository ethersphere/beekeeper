package check

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options ...
type Options struct {
	NodeCount       int
	NodeURLTemplate string
}

// Nodes ...
func Nodes(o Options) (err error) {
	var nodes []bee.Node
	for i := 0; i < o.NodeCount; i++ {
		node, err := bee.NewNode(fmt.Sprintf(o.NodeURLTemplate, i))
		if err != nil {
			return err
		}

		nodes = append(nodes, *node)
	}

	for i, n := range nodes {
		fmt.Printf("Node %d:\n", i)
		printNode(n)
	}

	return
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

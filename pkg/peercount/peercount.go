package peercount

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

// Options TODO
type Options struct {
	NodeCount       int
	NodeURLTemplate string
}

// BeeNode TODO
type BeeNode struct {
	Addresses BeeAddresses
	Peers     BeePeers
}

// BeeAddresses TODO
type BeeAddresses struct {
	Overlay  string   `json:"overlay"`
	Underlay []string `json:"underlay"`
}

// BeePeers TODO
type BeePeers struct {
	Peers []BeePeer `json:"peers"`
}

// BeePeer TODO
type BeePeer struct {
	Address string `json:"address"`
}

// Check TODO
func Check(o Options) {
	var nodes []BeeNode
	for i := 0; i < o.NodeCount; i++ {
		node, err := getNode(fmt.Sprintf(o.NodeURLTemplate, i))
		if err != nil {
			log.Fatalln(err)
		}

		nodes = append(nodes, node)

		fmt.Printf("Node %d:\n", i)
		printNode(node)
	}
}

func getNode(nodeURL string) (node BeeNode, err error) {
	// get addresses
	err = request(http.MethodGet, nodeURL+"/addresses", nil, &node.Addresses)
	if err != nil {
		return BeeNode{}, err
	}

	// get peers
	err = request(http.MethodGet, nodeURL+"/peers", nil, &node.Peers)
	if err != nil {
		return BeeNode{}, err
	}

	return
}

func printNode(node BeeNode) {
	fmt.Println(fmt.Sprintf("Overlay: %v", node.Addresses.Overlay))
	for _, u := range node.Addresses.Underlay {
		fmt.Println(fmt.Sprintf("Underlay: %v", u))
	}
	for _, p := range node.Peers.Peers {
		fmt.Println(fmt.Sprintf("Peer: %v", p.Address))
	}
}

var errPeerCount = errors.New("peer count")

func testPeerCount(nodes []BeeNode, peerCount int) error {
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

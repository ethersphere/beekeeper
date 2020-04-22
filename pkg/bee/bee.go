package bee

import (
	"net/http"
)

// Node ...
type Node struct {
	DebugURL  string
	Addresses Addresses
	Peers     Peers
}

// NewNode ...
func NewNode(DebugURL string) (node *Node, err error) {
	node = &Node{DebugURL: DebugURL}
	if err = node.addresses(); err != nil {
		return &Node{}, err
	}

	if err = node.peers(); err != nil {
		return &Node{}, err
	}

	return
}

// Addresses ...
type Addresses struct {
	Overlay  string   `json:"overlay"`
	Underlay []string `json:"underlay"`
}

// Peers ...
type Peers struct {
	Peers []Peer `json:"peers"`
}

// Peer ...
type Peer struct {
	Address string `json:"address"`
}

func (n *Node) addresses() (err error) {
	return request(http.MethodGet, n.DebugURL+"/addresses", nil, &n.Addresses)
}

func (n *Node) peers() (err error) {
	return request(http.MethodGet, n.DebugURL+"/peers", nil, &n.Peers)
}

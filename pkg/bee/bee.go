package bee

import "net/http"

// Node ...
type Node struct {
	DebugURL  string
	Addresses Addresses
	Peers     Peers
}

// NewNode ...
func NewNode(DebugURL string) (node *Node, err error) {
	addresses, err := getAddresses(DebugURL)
	if err != nil {
		return nil, err
	}

	peers, err := getPeers(DebugURL)
	if err != nil {
		return nil, err
	}

	return &Node{
		DebugURL:  DebugURL,
		Addresses: addresses,
		Peers:     peers,
	}, nil
}

// Addresses ...
type Addresses struct {
	Overlay  string   `json:"overlay"`
	Underlay []string `json:"underlay"`
}

// getAddresses ...
func getAddresses(nodeURL string) (addresses Addresses, err error) {
	err = request(http.MethodGet, nodeURL+"/addresses", nil, &addresses)
	if err != nil {
		return Addresses{}, err
	}

	return
}

// Peers ...
type Peers struct {
	Peers []Peer `json:"peers"`
}

// Peer ...
type Peer struct {
	Address string `json:"address"`
}

// // getPeers ...
func getPeers(nodeURL string) (peers Peers, err error) {
	err = request(http.MethodGet, nodeURL+"/peers", nil, &peers)
	if err != nil {
		return Peers{}, err
	}

	return
}

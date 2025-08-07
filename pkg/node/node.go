package node

import (
	"github.com/ethersphere/beekeeper/pkg/bee/api"
)

type Node struct {
	client *api.Client
	name   string
}

type NodeList []Node

func NewNode(client *api.Client, name string) *Node {
	return &Node{
		client: client,
		name:   name,
	}
}

func (n *Node) Name() string {
	return n.name
}

func (n *Node) Client() *api.Client {
	return n.client
}

func (ns NodeList) Get(name string) *Node {
	for _, n := range ns {
		if n.Name() == name {
			return &n
		}
	}
	return nil
}

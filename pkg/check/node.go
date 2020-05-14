package check

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
)

// Node represents Bee node
type Node struct {
	a *api.Client
	d *debugapi.Client
}

// NewNode returns Bee node
func NewNode(APIHostnamePattern, APINamespace, APIDomain, DebugAPIHostnamePattern, DebugAPINamespace, DebugAPIDomain string, index int, disableNamespace bool) (node Node, err error) {
	APIURL, err := createURL(scheme, APIHostnamePattern, APINamespace, APIDomain, index, disableNamespace)
	if err != nil {
		return Node{}, err
	}
	debugAPIURL, err := createURL(scheme, DebugAPIHostnamePattern, DebugAPINamespace, DebugAPIDomain, index, disableNamespace)
	if err != nil {
		return Node{}, err
	}

	node = Node{
		a: api.NewClient(APIURL, nil),
		d: debugapi.NewClient(debugAPIURL, nil),
	}

	return
}

// NewNNodes returns N Bee Nodes
func NewNNodes(APIHostnamePattern, APINamespace, APIDomain, DebugAPIHostnamePattern, DebugAPINamespace, DebugAPIDomain string, disableNamespace bool, count int) (nodes []Node, err error) {
	for i := 0; i < count; i++ {
		n, err := NewNode(APIHostnamePattern, APINamespace, APIDomain, DebugAPIHostnamePattern, DebugAPINamespace, DebugAPIDomain, i, disableNamespace)
		if err != nil {
			return []Node{}, err
		}

		ctx := context.Background()
		a, err := n.d.Node.Addresses(ctx)
		if err != nil {
			return []Node{}, err
		}
		fmt.Println(i, a.Overlay)

		nodes = append(nodes, n)
	}

	for i, n := range nodes {
		ctx := context.Background()
		a, err := n.d.Node.Addresses(ctx)
		if err != nil {
			return []Node{}, err
		}
		fmt.Println(i, a.Overlay)
	}

	return
}

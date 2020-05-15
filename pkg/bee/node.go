package bee

import (
	"fmt"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
)

const (
	scheme = "http"
)

// Node represents Bee node
type Node struct {
	API      *api.Client
	DebugAPI *debugapi.Client
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
		API:      api.NewClient(APIURL, nil),
		DebugAPI: debugapi.NewClient(debugAPIURL, nil),
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

		nodes = append(nodes, n)
	}

	return
}

// createURL creates API or debug API URL
func createURL(scheme, hostnamePattern, namespace, domain string, counter int, disableNamespace bool) (nodeURL *url.URL, err error) {
	hostname := fmt.Sprintf(hostnamePattern, counter)
	if disableNamespace {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s", scheme, hostname, domain))
	} else {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", scheme, hostname, namespace, domain))
	}
	return
}

package node

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

type DeploymentType string

const (
	DeploymentTypeBeekeeper DeploymentType = "beekeeper"
	DeploymentTypeHelm      DeploymentType = "helm"
)

type NodeProvider interface {
	GetNodes(ctx context.Context) (NodeList, error)
	Namespace() string
}

type ClientConfig struct {
	Log            logging.Logger
	HTTPClient     *http.Client
	K8sClient      *k8s.Client
	BeeClients     map[string]*bee.Client
	Namespace      string
	LabelSelector  string
	DeploymentType DeploymentType
	InCluster      bool
	UseNamespace   bool     // Overrides the usage of the bee clients
	NodeGroups     []string // Node groups for filtering nodes (only used with beekeeper deployment)
}

type Client struct {
	log            logging.Logger
	httpClient     *http.Client
	k8sClient      *k8s.Client
	beeClients     map[string]*bee.Client
	namespace      string
	labelSelector  string
	deploymentType DeploymentType
	inCluster      bool
	useNamespace   bool
	nodeGroups     []string
}

func New(cfg *ClientConfig) *Client {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}

	return &Client{
		log:            cfg.Log,
		httpClient:     cfg.HTTPClient,
		k8sClient:      cfg.K8sClient,
		beeClients:     cfg.BeeClients,
		namespace:      cfg.Namespace,
		labelSelector:  cfg.LabelSelector,
		deploymentType: cfg.DeploymentType,
		inCluster:      cfg.InCluster,
		useNamespace:   cfg.UseNamespace,
		nodeGroups:     cfg.NodeGroups,
	}
}

func (sc *Client) Namespace() string {
	return sc.namespace
}

func (sc *Client) GetNodes(ctx context.Context) (nodes NodeList, err error) {
	defer func() {
		if err != nil || nodes == nil {
			return
		}
		for i := range nodes {
			sc.log.Debugf("adding node %s with endpoint %s", nodes[i].Name(), nodes[i].Client().Host())
		}
		sc.log.Infof("found %d nodes", len(nodes))
	}()

	if sc.useNamespace && sc.namespace != "" {
		nodes, err = sc.getNamespaceNodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("get namespace nodes: %w", err)
		}
		return
	}

	if len(sc.beeClients) == 0 {
		return nil, fmt.Errorf("bee clients not provided")
	}

	// Filter bee clients by node groups if specified
	filteredClients := sc.beeClients
	if len(sc.nodeGroups) > 0 {
		filteredClients = sc.filterClientsByNodeGroups(sc.beeClients)
		if len(filteredClients) == 0 {
			sc.log.Warningf("no nodes found in specified node groups: %v", sc.nodeGroups)
			return NodeList{}, nil
		}
		sc.log.Debugf("filtered to %d nodes from node groups: %v", len(filteredClients), sc.nodeGroups)
	}

	names := make([]string, 0, len(filteredClients))
	for n := range filteredClients {
		names = append(names, n)
	}
	sort.Strings(names)

	nodes = make(NodeList, 0, len(names))
	for _, nodeName := range names {
		beeClient := filteredClients[nodeName]
		nodes = append(nodes, *NewNode(beeClient.API(), sc.nodeName(nodeName)))
	}

	return nodes, nil
}

func (sc *Client) getNamespaceNodes(ctx context.Context) (nodes []Node, err error) {
	if sc.namespace == "" {
		return nil, fmt.Errorf("namespace not provided")
	}

	if sc.k8sClient == nil {
		return nil, fmt.Errorf("k8s client not provided")
	}

	if sc.inCluster {
		nodes, err = sc.getServiceNodes(ctx)
	} else {
		nodes, err = sc.getIngressNodes(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("get nodes: %w", err)
	}

	return nodes, nil
}

func (sc *Client) getServiceNodes(ctx context.Context) ([]Node, error) {
	svcNodes, err := sc.k8sClient.Service.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list api services: %w", err)
	}

	nodes := make([]Node, len(svcNodes))
	for i, node := range svcNodes {
		parsedURL, err := url.Parse(node.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("extract base URL: %w", err)
		}

		apiClient, err := api.NewClient(parsedURL, sc.httpClient)
		if err != nil {
			return nil, fmt.Errorf("create api client: %w", err)
		}

		nodes[i] = *NewNode(apiClient, sc.nodeName(node.Name))
	}

	return nodes, nil
}

func (sc *Client) getIngressNodes(ctx context.Context) ([]Node, error) {
	ingressNodes, err := sc.k8sClient.Ingress.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list ingress api nodes hosts: %w", err)
	}

	ingressRouteNodes, err := sc.k8sClient.IngressRoute.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list ingress route api nodes hosts: %w", err)
	}

	allNodes := append(ingressNodes, ingressRouteNodes...)
	nodes := make([]Node, len(allNodes))
	for i, node := range allNodes {
		apiURL, err := url.Parse(fmt.Sprintf("http://%s", node.Host))
		if err != nil {
			return nil, fmt.Errorf("extract base URL: %w", err)
		}

		apiClient, err := api.NewClient(apiURL, sc.httpClient)
		if err != nil {
			return nil, fmt.Errorf("create api client: %w", err)
		}

		nodes[i] = *NewNode(apiClient, sc.nodeName(node.Name))
	}

	return nodes, nil
}

// filterClientsByNodeGroups filters bee clients by node groups
// Only applies filtering for beekeeper deployments where NodeGroupName is set
func (sc *Client) filterClientsByNodeGroups(clients map[string]*bee.Client) map[string]*bee.Client {
	filteredClients := make(map[string]*bee.Client)
	if len(sc.nodeGroups) > 0 && sc.deploymentType == DeploymentTypeBeekeeper {
		for _, ng := range sc.nodeGroups {
			for name, client := range clients {
				if client.NodeGroup() == ng {
					filteredClients[name] = client
				}
			}
		}
	} else {
		filteredClients = clients
	}
	return filteredClients
}

// nodeName returns the name of the node, and adds suffix based on deployment type.
// In case of Beekeeper deployment, it adds "-0" suffix, because there is only one replica.
// In case of other deployments, it returns the name as is.
func (sc *Client) nodeName(name string) string {
	if sc.deploymentType == DeploymentTypeBeekeeper {
		return fmt.Sprintf("%s-0", name)
	}
	return name
}

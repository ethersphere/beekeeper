package node

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

type NodeProvider interface {
	GetNodes(ctx context.Context) (NodeList, error)
}

type ClientConfig struct {
	Log           logging.Logger
	Namespace     string
	HTTPClient    *http.Client
	K8sClient     *k8s.Client
	BeeClients    map[string]*bee.Client
	LabelSelector string
	InCluster     bool
}

type Client struct {
	log           logging.Logger
	namespace     string
	k8sClient     *k8s.Client
	httpClient    *http.Client
	beeClients    map[string]*bee.Client
	labelSelector string
	inCluster     bool
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
		log:           cfg.Log,
		namespace:     cfg.Namespace,
		k8sClient:     cfg.K8sClient,
		beeClients:    cfg.BeeClients,
		labelSelector: cfg.LabelSelector,
		inCluster:     cfg.InCluster,
		httpClient:    cfg.HTTPClient,
	}
}

func (sc *Client) GetNodes(ctx context.Context) (nodes NodeList, err error) {
	if sc.namespace != "" {
		return sc.getNamespaceNodes(ctx)
	}

	if sc.beeClients == nil {
		return nil, fmt.Errorf("bee clients not provided")
	}

	nodes = make(NodeList, 0, len(sc.beeClients))
	for nodeName, beeClient := range sc.beeClients {
		nodes = append(nodes, *NewNode(beeClient.API(), nodeName))
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

		nodes[i] = *NewNode(apiClient, node.Name)
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

		nodes[i] = *NewNode(apiClient, node.Name)
	}

	return nodes, nil
}

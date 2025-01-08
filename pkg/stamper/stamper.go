package stamper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

type Client interface {
	Create(ctx context.Context, amount uint64, depth uint8) error
	Dilute(ctx context.Context, threshold float64, depth uint16) error
	Set(ctx context.Context, ttlThreshold, topupDuration time.Duration, threshold float64, depth uint16) error
	Topup(ctx context.Context, ttlThreshold, topupDuration time.Duration) error
}

type ClientConfig struct {
	Log           logging.Logger
	Namespace     string
	K8sClient     *k8s.Client
	HTTPClient    *http.Client // injected HTTP client
	LabelSelector string
	InCluster     bool
}

type StamperClient struct {
	log           logging.Logger
	namespace     string
	k8sClient     *k8s.Client
	labelSelector string
	inCluster     bool
	httpClient    http.Client
}

func NewStamperClient(cfg *ClientConfig) *StamperClient {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	// use the injected HTTP client if available, else create a new one
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &StamperClient{
		httpClient:    *httpClient,
		log:           cfg.Log,
		namespace:     cfg.Namespace,
		k8sClient:     cfg.K8sClient,
		labelSelector: cfg.LabelSelector,
		inCluster:     cfg.InCluster,
	}
}

// Create implements Client.
func (s *StamperClient) Create(ctx context.Context, amount uint64, depth uint8) error {
	panic("unimplemented")
}

// Dilute implements Client.
func (s *StamperClient) Dilute(ctx context.Context, usageThreshold float64, dilutionDepth uint16) error {
	s.log.WithFields(map[string]interface{}{"usageThreshold": usageThreshold, "dilutionDepth": dilutionDepth}).Infof("diluting namespace %s", s.namespace)
	nodes, err := s.getNamespaceNodes(ctx)
	if err != nil {
		return fmt.Errorf("get namespace nodes: %w", err)
	}

	for _, node := range nodes {
		if err := node.Dilute(ctx, usageThreshold, dilutionDepth); err != nil {
			return fmt.Errorf("dilute node %s: %w", node.Name, err)
		}
	}

	return nil
}

// Set implements Client.
func (s *StamperClient) Set(ctx context.Context, ttlThreshold time.Duration, topupTo time.Duration, usageThreshold float64, dilutionDepth uint16) error {
	panic("unimplemented")
}

// Topup implements Client.
func (s *StamperClient) Topup(ctx context.Context, ttlThreshold time.Duration, topupTo time.Duration) (err error) {
	nodes, err := s.getNamespaceNodes(ctx)
	if err != nil {
		return fmt.Errorf("get namespace nodes: %w", err)
	}

	for _, node := range nodes {
		_ = node
		// do something with node
	}

	return nil
}

func (sc *StamperClient) getNamespaceNodes(ctx context.Context) (nodes []Node, err error) {
	if sc.namespace == "" {
		return nil, fmt.Errorf("namespace not provided")
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

func (sc *StamperClient) getServiceNodes(ctx context.Context) ([]Node, error) {
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

		apiClient := api.NewClient(parsedURL, &api.ClientOptions{
			HTTPClient: &sc.httpClient,
		})

		nodes[i] = *NewNodeInfo(apiClient, node.Name)
	}

	return nodes, nil
}

func (sc *StamperClient) getIngressNodes(ctx context.Context) ([]Node, error) {
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
		parsedURL, err := url.Parse(fmt.Sprintf("http://%s", node.Host))
		if err != nil {
			return nil, fmt.Errorf("extract base URL: %w", err)
		}

		apiClient := api.NewClient(parsedURL, &api.ClientOptions{
			HTTPClient: &sc.httpClient,
		})

		nodes[i] = *NewNodeInfo(apiClient, node.Name)
	}

	return nodes, nil
}

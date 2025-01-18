package stamper

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/swap"
	"golang.org/x/sync/errgroup"
)

// Client is the interface for stamper client.
type Client interface {
	// Create creates a postage batch.
	Create(ctx context.Context, amount uint64, depth uint16) error
	// Dilute dilutes a postage batch.
	Dilute(ctx context.Context, threshold float64, depth uint16, opts ...Option) error
	// Set sets the topup and dilution parameters.
	Set(ctx context.Context, ttlThreshold, topupDuration time.Duration, threshold float64, depth uint16, opts ...Option) error
	// Topup tops up a postage batch.
	Topup(ctx context.Context, ttlThreshold, topupDuration time.Duration, opts ...Option) error
}

type Option func(*options)

type options struct {
	batchIds []string
}

func WithBatchIDs(batchIds []string) Option {
	return func(o *options) {
		o.batchIds = batchIds
	}
}

type ClientConfig struct {
	Log           logging.Logger
	Namespace     string
	K8sClient     *k8s.Client
	SwapClient    swap.Client
	BeeClients    map[string]*bee.Client
	LabelSelector string
	InCluster     bool
}

type StamperClient struct {
	log           logging.Logger
	namespace     string
	k8sClient     *k8s.Client
	swapClient    swap.Client
	beeClients    map[string]*bee.Client
	labelSelector string
	inCluster     bool
}

func New(cfg *ClientConfig) *StamperClient {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	return &StamperClient{
		log:           cfg.Log,
		namespace:     cfg.Namespace,
		k8sClient:     cfg.K8sClient,
		swapClient:    cfg.SwapClient,
		beeClients:    cfg.BeeClients,
		labelSelector: cfg.LabelSelector,
		inCluster:     cfg.InCluster,
	}
}

// Create implements Client.
func (s *StamperClient) Create(ctx context.Context, amount uint64, depth uint16) error {
	s.log.WithFields(map[string]interface{}{
		"amount": amount,
		"depth":  depth,
	}).Infof("creating postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for _, node := range nodes {
		g.TryGo(func() error {
			return node.Create(ctx, amount, depth)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("create postage batch: %w", err)
	}

	return nil
}

// Dilute implements Client.
func (s *StamperClient) Dilute(ctx context.Context, usageThreshold float64, dilutionDepth uint16, opts ...Option) error {
	s.log.WithFields(map[string]interface{}{
		"usageThreshold": usageThreshold,
		"dilutionDepth":  dilutionDepth,
	}).Infof("diluting postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for _, node := range nodes {
		g.TryGo(func() error {
			return node.Dilute(ctx, usageThreshold, dilutionDepth, processOptions(opts...).batchIds)
		})
	}

	if err := g.Wait(); err != nil {
		s.log.Errorf("dilute postage batch: %v", err)
	}

	return nil
}

// Set implements Client.
func (s *StamperClient) Set(ctx context.Context, ttlThreshold time.Duration, topupTo time.Duration, usageThreshold float64, dilutionDepth uint16, opts ...Option) error {
	s.log.WithFields(map[string]interface{}{
		"ttlThreshold":   ttlThreshold,
		"topupTo":        topupTo,
		"usageThreshold": usageThreshold,
		"dilutionDepth":  dilutionDepth,
	}).Infof("setting topup and dilution on postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	blockTime, err := s.swapClient.FetchBlockTime(ctx)
	if err != nil {
		return fmt.Errorf("fetching block time: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for _, node := range nodes {
		g.TryGo(func() error {
			return node.Set(ctx, ttlThreshold, topupTo, usageThreshold, dilutionDepth, blockTime, processOptions(opts...).batchIds)
		})
	}

	if err := g.Wait(); err != nil {
		s.log.Errorf("set postage batch: %v", err)
	}

	return nil
}

// Topup implements Client.
func (s *StamperClient) Topup(ctx context.Context, ttlThreshold time.Duration, topupTo time.Duration, opts ...Option) (err error) {
	s.log.WithFields(map[string]interface{}{
		"ttlThreshold": ttlThreshold,
		"topupTo":      topupTo,
	}).Infof("topup postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	blockTime, err := s.swapClient.FetchBlockTime(ctx)
	if err != nil {
		return fmt.Errorf("fetching block time: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for _, node := range nodes {
		g.TryGo(func() error {
			return node.Topup(ctx, ttlThreshold, topupTo, blockTime, processOptions(opts...).batchIds)
		})
	}

	if err := g.Wait(); err != nil {
		s.log.Errorf("topup postage batch: %v", err)
	}

	return nil
}

func (sc *StamperClient) getNodes(ctx context.Context) (nodes []node, err error) {
	if sc.namespace != "" {
		return sc.getNamespaceNodes(ctx)
	}

	if sc.beeClients == nil {
		return nil, fmt.Errorf("bee clients not provided")
	}

	nodes = make([]node, 0, len(sc.beeClients))
	for nodeName, beeClient := range sc.beeClients {
		nodes = append(nodes, *newNodeInfo(beeClient.API(), nodeName, sc.log))
	}

	return nodes, nil
}

func (sc *StamperClient) getNamespaceNodes(ctx context.Context) (nodes []node, err error) {
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

func (sc *StamperClient) getServiceNodes(ctx context.Context) ([]node, error) {
	svcNodes, err := sc.k8sClient.Service.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list api services: %w", err)
	}

	nodes := make([]node, len(svcNodes))
	for i, node := range svcNodes {
		parsedURL, err := url.Parse(node.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("extract base URL: %w", err)
		}

		apiClient := api.NewClient(parsedURL, nil)

		nodes[i] = *newNodeInfo(apiClient, node.Name, sc.log)
	}

	return nodes, nil
}

func (sc *StamperClient) getIngressNodes(ctx context.Context) ([]node, error) {
	ingressNodes, err := sc.k8sClient.Ingress.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list ingress api nodes hosts: %w", err)
	}

	ingressRouteNodes, err := sc.k8sClient.IngressRoute.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list ingress route api nodes hosts: %w", err)
	}

	allNodes := append(ingressNodes, ingressRouteNodes...)
	nodes := make([]node, len(allNodes))
	for i, node := range allNodes {
		parsedURL, err := url.Parse(fmt.Sprintf("http://%s", node.Host))
		if err != nil {
			return nil, fmt.Errorf("extract base URL: %w", err)
		}

		apiClient := api.NewClient(parsedURL, nil)

		nodes[i] = *newNodeInfo(apiClient, node.Name, sc.log)
	}

	return nodes, nil
}

func processOptions(opts ...Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

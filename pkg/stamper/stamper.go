package stamper

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/ethersphere/bee/pkg/postage"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/swap"
)

type Option func(*options)

type options struct {
	batchIDs      []string
	postageLabels []string
}

func WithBatchIDs(batchIds []string) Option {
	return func(o *options) {
		o.batchIDs = batchIds
	}
}

func WithPostageLabels(postageLabels []string) Option {
	return func(o *options) {
		o.postageLabels = postageLabels
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

type Client struct {
	log           logging.Logger
	namespace     string
	k8sClient     *k8s.Client
	swapClient    swap.Client
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

	return &Client{
		log:           cfg.Log,
		namespace:     cfg.Namespace,
		k8sClient:     cfg.K8sClient,
		swapClient:    cfg.SwapClient,
		beeClients:    cfg.BeeClients,
		labelSelector: cfg.LabelSelector,
		inCluster:     cfg.InCluster,
	}
}

// Create creates a postage batch.
func (s *Client) Create(ctx context.Context, duration time.Duration, depth uint16, postageLabel string) error {
	if duration == 0 {
		return fmt.Errorf("duration must be greater than 0")
	}

	if depth <= postage.BucketDepth {
		return fmt.Errorf("depth must be greater than %d", postage.BucketDepth)
	}

	s.log.WithFields(map[string]interface{}{
		"duration": duration,
		"depth":    depth,
	}).Infof("creating postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	blockTime, err := s.swapClient.FetchBlockTime(ctx)
	if err != nil {
		return fmt.Errorf("fetching block time: %w", err)
	}

	for _, node := range nodes {
		if err := node.Create(ctx, duration, depth, postageLabel, blockTime); err != nil {
			s.log.Errorf("node %s create postage batch: %v", node.name, err)
		}
	}

	return nil
}

// Dilute dilutes a postage batch.
func (s *Client) Dilute(ctx context.Context, usageThreshold float64, dilutionDepth uint16, opts ...Option) error {
	s.log.WithFields(map[string]interface{}{
		"usageThreshold": usageThreshold,
		"dilutionDepth":  dilutionDepth,
	}).Infof("diluting postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	for _, node := range nodes {
		if err := node.Dilute(ctx, usageThreshold, dilutionDepth, processOptions(opts...)); err != nil {
			s.log.Errorf("node %s dilute postage batch: %v", node.name, err)
		}
	}

	return nil
}

// Set sets the topup and dilution parameters.
func (s *Client) Set(ctx context.Context, ttlThreshold time.Duration, topupTo time.Duration, usageThreshold float64, dilutionDepth uint16, opts ...Option) error {
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

	for _, node := range nodes {
		if err := node.Set(ctx, ttlThreshold, topupTo, usageThreshold, dilutionDepth, blockTime, processOptions(opts...)); err != nil {
			s.log.Errorf("node %s set postage batch: %v", node.name, err)
		}
	}

	return nil
}

// Topup tops up a postage batch.
func (s *Client) Topup(ctx context.Context, ttlThreshold time.Duration, topupTo time.Duration, opts ...Option) (err error) {
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

	for _, node := range nodes {
		if err := node.Topup(ctx, ttlThreshold, topupTo, blockTime, processOptions(opts...)); err != nil {
			s.log.Errorf("node %s topup postage batch: %v", node.name, err)
		}
	}

	return nil
}

func (sc *Client) getNodes(ctx context.Context) (nodes []node, err error) {
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

func (sc *Client) getNamespaceNodes(ctx context.Context) (nodes []node, err error) {
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

func (sc *Client) getServiceNodes(ctx context.Context) ([]node, error) {
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

func (sc *Client) getIngressNodes(ctx context.Context) ([]node, error) {
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

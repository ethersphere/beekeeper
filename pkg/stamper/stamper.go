package stamper

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethersphere/bee/v2/pkg/postage"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/node"
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
	Log        logging.Logger
	SwapClient swap.BlockTimeFetcher
	NodeClient node.NodeProvider
}

type Client struct {
	log        logging.Logger
	nodeClient node.NodeProvider
	swapClient swap.BlockTimeFetcher
}

func New(cfg *ClientConfig) *Client {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	if cfg.SwapClient == nil {
		cfg.SwapClient = &swap.NotSet{}
	}

	if cfg.NodeClient == nil {
		cfg.NodeClient = &node.NotSet{}
	}

	return &Client{
		log:        cfg.Log,
		nodeClient: cfg.NodeClient,
		swapClient: cfg.SwapClient,
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
	}).Info("creating postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	blockTime, err := s.swapClient.FetchBlockTime(ctx, swap.WithOffset(1000))
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
	}).Info("diluting postage batch on nodes")

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
	}).Info("setting topup and dilution on postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	blockTime, err := s.swapClient.FetchBlockTime(ctx, swap.WithOffset(1000))
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
	}).Info("topup postage batch on nodes")

	nodes, err := s.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("get nodes: %w", err)
	}

	blockTime, err := s.swapClient.FetchBlockTime(ctx, swap.WithOffset(1000))
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

func (s *Client) getNodes(ctx context.Context) (nodes []stamperNode, err error) {
	nodeList, err := s.nodeClient.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("get nodes: %w", err)
	}

	nodes = make([]stamperNode, len(nodeList))
	for i, n := range nodeList {
		nodes[i] = *newStamperNode(n.Client(), n.Name(), s.log)
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

package node

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/node"
	"github.com/ethersphere/node-funder/pkg/funder"
)

var _ funder.NodeLister = (*Client)(nil)

type Client struct {
	nodeProvider node.NodeProvider
	log          logging.Logger
}

func NewClient(nodeProvider node.NodeProvider, l logging.Logger) *Client {
	return &Client{
		nodeProvider: nodeProvider,
		log:          l,
	}
}

func (c *Client) List(ctx context.Context, _ string) ([]funder.NodeInfo, error) {
	nodes, err := c.nodeProvider.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list api services: %w", err)
	}

	nodeInfos := make([]funder.NodeInfo, len(nodes))
	for i, node := range nodes {
		nodeInfos[i] = funder.NodeInfo{
			Name:    node.Name(),
			Address: node.Client().Host(),
		}
	}

	return nodeInfos, nil
}

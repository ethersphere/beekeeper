package node

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/node-funder/pkg/funder"
)

type Client struct {
	k8sClient *k8s.Client
	inCluster bool
	label     string
	log       logging.Logger
}

func NewClient(k8sClient *k8s.Client, inCluster bool, label string, l logging.Logger) *Client {
	return &Client{
		k8sClient: k8sClient,
		label:     label,
		log:       l,
	}
}

func (c *Client) List(ctx context.Context, namespace string) ([]funder.NodeInfo, error) {
	if c.k8sClient == nil {
		return nil, fmt.Errorf("k8s client not initialized")
	}

	if namespace == "" {
		return nil, fmt.Errorf("namespace not provided")
	}

	if c.inCluster {
		return c.getServiceNodes(ctx, namespace)
	}

	return c.getIngressNodes(ctx, namespace)
}

func (c *Client) getServiceNodes(ctx context.Context, namespace string) ([]funder.NodeInfo, error) {
	svcNodes, err := c.k8sClient.Service.GetNodes(ctx, namespace, c.label)
	if err != nil {
		return nil, fmt.Errorf("list api services: %w", err)
	}

	nodes := make([]funder.NodeInfo, len(svcNodes))
	for i, node := range svcNodes {
		nodes[i] = funder.NodeInfo{
			Name:    node.Name,
			Address: node.Endpoint,
		}
	}

	return nodes, nil
}

func (c *Client) getIngressNodes(ctx context.Context, namespace string) ([]funder.NodeInfo, error) {
	ingressNodes, err := c.k8sClient.Ingress.GetNodes(ctx, namespace, c.label)
	if err != nil {
		return nil, fmt.Errorf("list ingress api nodes hosts: %w", err)
	}

	ingressRouteNodes, err := c.k8sClient.IngressRoute.GetIngressHosts(ctx, namespace, c.label)
	if err != nil {
		return nil, fmt.Errorf("list ingress route api nodes hosts: %w", err)
	}

	allNodes := append(ingressNodes, ingressRouteNodes...)
	nodes := make([]funder.NodeInfo, len(allNodes))
	for i, node := range allNodes {
		nodes[i] = funder.NodeInfo{
			Name:    node.Name,
			Address: fmt.Sprintf("http://%s", node.Host),
		}
	}

	return nodes, nil
}

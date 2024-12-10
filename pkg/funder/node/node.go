package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/node-funder/pkg/funder"
)

type Client struct {
	k8sClient *k8s.Client
	label     string
	log       logging.Logger
}

func NewClient(k8sClient *k8s.Client, label string, l logging.Logger) *Client {
	return &Client{
		k8sClient: k8sClient,
		label:     label,
		log:       l,
	}
}

func (c *Client) List(ctx context.Context, namespace string) (nodes []funder.NodeInfo, err error) {
	if c.k8sClient == nil {
		return nil, fmt.Errorf("k8s client not initialized")
	}

	if namespace == "" {
		return nil, fmt.Errorf("namespace not provided")
	}

	ingressHosts, err := c.k8sClient.Ingress.GetIngressHosts(ctx, namespace, c.label)
	if err != nil {
		return nil, fmt.Errorf("list ingress api nodes hosts: %s", err.Error())
	}

	ingressRouteHosts, err := c.k8sClient.IngressRoute.GetIngressHosts(ctx, namespace, c.label)
	if err != nil {
		return nil, fmt.Errorf("list ingress route api nodes hosts: %s", err.Error())
	}

	ingressHosts = append(ingressHosts, ingressRouteHosts...)

	for _, node := range ingressHosts {
		nodes = append(nodes, funder.NodeInfo{
			Name:    strings.TrimSuffix(node.Name, "-api"),
			Address: fmt.Sprintf("http://%s", node.Host),
		})
	}

	return nodes, nil
}

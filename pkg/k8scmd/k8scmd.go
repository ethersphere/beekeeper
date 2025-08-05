package k8scmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

type ClientConfig struct {
	Log        logging.Logger
	K8sClient  *k8s.Client
	BeeClients map[string]*bee.Client
}

type Client struct {
	log        logging.Logger
	k8sClient  *k8s.Client
	beeClients map[string]*bee.Client
}

func New(cfg *ClientConfig) *Client {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	return &Client{
		log:        cfg.Log,
		k8sClient:  cfg.K8sClient,
		beeClients: cfg.BeeClients,
	}
}

// Run sends a update command to the Bee clients in the Kubernetes cluster
func (c *Client) Run(ctx context.Context, namespace, labelSelector string, args []string) (statefulSets []string, err error) {
	c.log.Info("updating Bee cluster")

	if namespace == "" {
		return nil, errors.New("namespace cannot be empty")
	}

	if len(args) == 0 {
		return nil, errors.New("args cannot be empty")
	}

	if len(c.beeClients) == 0 {
		statefulSets, err = c.k8sClient.StatefulSet.RunningStatefulSets(ctx, namespace, labelSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to get running stateful sets in namespace %s: %w", namespace, err)
		}
	} else {
		for _, client := range c.beeClients {
			statefulSets = append(statefulSets, client.Name())
			statefulSet, err := c.k8sClient.Pods.GetControllingStatefulSet(ctx, client.Name(), namespace)
			if err != nil {
				c.log.Warning("failed to get controlling StatefulSet for Bee client %s: %v", client.Name(), err)
				continue
			}
			statefulSets = append(statefulSets, statefulSet)
		}
	}

	c.log.Debugf("found %d stateful sets to update", len(statefulSets))

	for _, ss := range statefulSets {
		c.log.Debugf("changing stateful set cmd to update: %s", ss)
		if err := c.k8sClient.StatefulSet.UpdateCommand(ctx, namespace, ss, args); err != nil {
			return nil, err
		}
	}

	c.log.Info("updating Bee cluster completed")

	return statefulSets, nil
}

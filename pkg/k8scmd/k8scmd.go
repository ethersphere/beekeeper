package k8scmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
func (c *Client) Run(ctx context.Context, namespace, labelSelector string, args []string) (err error) {
	c.log.Info("updating Bee cluster")

	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}

	if len(args) == 0 {
		return errors.New("args cannot be empty")
	}

	var statefulSets []string
	statefulSetsMap := make(map[string]struct{})

	if len(c.beeClients) == 0 {
		statefulSets, err = c.k8sClient.StatefulSet.RunningStatefulSets(ctx, namespace, labelSelector)
		if err != nil {
			return fmt.Errorf("failed to get running stateful sets in namespace %s: %w", namespace, err)
		}
	} else {
		for _, client := range c.beeClients {
			statefulSet, err := c.k8sClient.Pods.GetControllingStatefulSet(ctx, client.Name(), namespace)
			if err != nil {
				if k8serrors.IsNotFound(err) {
					// this happenes when cluster is deployed by beekeeper, because each bee client (pod) has its own stateful set, with the same name
					c.log.Warning("failed to get controlling StatefulSet for Bee client %s: %v", client.Name(), err)
					statefulSets = append(statefulSets, client.Name())
					continue
				}
				return fmt.Errorf("failed to get controlling stateful set for Bee client %s in namespace %s: %w", client.Name(), namespace, err)
			}
			c.log.Debugf("found controlling stateful set for Bee client %s: %s", client.Name(), statefulSet)
			statefulSets = append(statefulSets, statefulSet)
		}
	}

	// remove duplicates from statefulSets
	for _, ss := range statefulSets {
		statefulSetsMap[ss] = struct{}{}
	}

	c.log.Debugf("found %d stateful sets to update", len(statefulSetsMap))

	for ss := range statefulSetsMap {
		c.log.Debugf("changing stateful set cmd to update: %s", ss)
		if err := c.k8sClient.StatefulSet.UpdateCommand(ctx, namespace, ss, args); err != nil {
			return fmt.Errorf("failed to update command for stateful set %s in namespace %s: %w", ss, namespace, err)
		}
	}

	c.log.Info("updating Bee cluster completed")

	return nil
}

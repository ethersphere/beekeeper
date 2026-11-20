package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/node"
	"github.com/spf13/cobra"
)

const optionNameDiscoveryType = "discovery-type"

func addDiscoveryTypeFlag(cmd *cobra.Command) {
	cmd.Flags().String(optionNameDiscoveryType, "", fmt.Sprintf(
		"How Bee nodes are discovered: '%s' (Beekeeper cluster config, requires --cluster-name), '%s' (services/ingresses matching --label-selector) or '%s' (pods of the StatefulSets listed in --%s). If not set, '%s' is used when --namespace is set, otherwise '%s'.",
		node.DiscoveryTypeBeeClients, node.DiscoveryTypeNamespace, node.DiscoveryTypeStatefulSet, optionNameStatefulSets,
		node.DiscoveryTypeNamespace, node.DiscoveryTypeBeeClients,
	))
}

func addStatefulSetsFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice(optionNameStatefulSets, nil, fmt.Sprintf(
		"List of StatefulSet names to discover nodes from (e.g., 'bootnode-0,bootnode-1'). Only used with --%s=%s.",
		optionNameDiscoveryType, node.DiscoveryTypeStatefulSet,
	))
}

type nodeClientConfig struct {
	namespace        string
	clusterName      string
	labelSelector    string
	deploymentType   string
	inCluster        bool
	nodeGroups       []string
	statefulSetNames []string
	discoveryType    node.DiscoveryType
}

func (c *command) createNodeClient(ctx context.Context, useDeploymentType bool) (*node.Client, error) {
	config, err := c.nodeClientConfig(useDeploymentType)
	if err != nil {
		return nil, err
	}

	if useDeploymentType {
		c.log.Infof("using deployment type %s", config.deploymentType)
	}

	var beeClients map[string]*bee.Client

	if config.clusterName != "" {
		cluster, err := c.setupCluster(ctx, config.clusterName, false)
		if err != nil {
			return nil, fmt.Errorf("setting up cluster %s: %w", config.clusterName, err)
		}

		beeClients, err = cluster.NodesClients(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve node clients: %w", err)
		}

		config.namespace = cluster.Namespace()
	}

	c.log.Infof("using discovery type %s", config.discoveryType)

	return node.New(&node.ClientConfig{
		Log:              c.log,
		HTTPClient:       c.httpClient,
		K8sClient:        c.k8sClient,
		BeeClients:       beeClients,
		Namespace:        config.namespace,
		LabelSelector:    config.labelSelector,
		DeploymentType:   node.DeploymentType(config.deploymentType),
		InCluster:        config.inCluster,
		DiscoveryType:    config.discoveryType,
		NodeGroups:       config.nodeGroups,
		StatefulSetNames: config.statefulSetNames,
	}), nil
}

// nodeClientConfig reads and validates the node client flags.
func (c *command) nodeClientConfig(useDeploymentType bool) (nodeClientConfig, error) {
	config := nodeClientConfig{
		namespace:        c.globalConfig.GetString(optionNameNamespace),
		clusterName:      c.globalConfig.GetString(optionNameClusterName),
		labelSelector:    c.globalConfig.GetString(optionNameLabelSelector),
		deploymentType:   c.globalConfig.GetString(optionNameDeploymentType),
		inCluster:        c.globalConfig.GetBool(optionNameInCluster),
		nodeGroups:       c.globalConfig.GetStringSlice(optionNameNodeGroups),
		statefulSetNames: c.globalConfig.GetStringSlice(optionNameStatefulSets),
	}

	if config.clusterName == "" && config.namespace == "" {
		return config, errors.New("either cluster name or namespace must be provided")
	}

	namespaceSet := c.globalConfig.IsSet(optionNameNamespace)

	if namespaceSet && config.namespace == "" {
		return config, errors.New("namespace cannot be empty if set")
	}

	if config.namespace == "" && useDeploymentType && !isValidDeploymentType(config.deploymentType) {
		return config, errors.New("unsupported deployment type: must be 'beekeeper' or 'helm'")
	}

	discoveryType, err := resolveDiscoveryType(
		node.DiscoveryType(c.globalConfig.GetString(optionNameDiscoveryType)),
		namespaceSet,
		config.clusterName,
		config.statefulSetNames,
	)
	if err != nil {
		return config, err
	}
	config.discoveryType = discoveryType

	return config, nil
}

// resolveDiscoveryType returns the requested discovery type. If none is
// requested, a set namespace takes precedence over the cluster bee clients.
func resolveDiscoveryType(requested node.DiscoveryType, namespaceSet bool, clusterName string, statefulSetNames []string) (node.DiscoveryType, error) {
	switch requested {
	case "":
		if namespaceSet {
			return node.DiscoveryTypeNamespace, nil
		}
		return node.DiscoveryTypeBeeClients, nil
	case node.DiscoveryTypeBeeClients:
		if clusterName == "" {
			return "", fmt.Errorf("discovery type %s requires --%s", requested, optionNameClusterName)
		}
		return requested, nil
	case node.DiscoveryTypeNamespace:
		return requested, nil
	case node.DiscoveryTypeStatefulSet:
		if len(statefulSetNames) == 0 {
			return "", fmt.Errorf("discovery type %s requires --%s", requested, optionNameStatefulSets)
		}
		return requested, nil
	default:
		return "", fmt.Errorf("unsupported discovery type %q: must be '%s', '%s' or '%s'",
			requested, node.DiscoveryTypeBeeClients, node.DiscoveryTypeNamespace, node.DiscoveryTypeStatefulSet)
	}
}

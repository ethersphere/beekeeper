package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/node"
)

func (c *command) createNodeClient(ctx context.Context, useDeploymentType bool) (*node.Client, error) {
	if err := c.validateNodeClientInputs(useDeploymentType); err != nil {
		return nil, err
	}

	config := c.extractNodeClientConfig()

	beeClients, namespace, err := c.setupBeeClients(ctx, config.clusterName)
	if err != nil {
		return nil, err
	}

	// Update namespace if we got it from cluster
	// Namespace is required for all K8s operations (pods, statefulsets, services, ingress)
	if namespace != "" {
		config.namespace = namespace
	}

	discoveryType := c.determineDiscoveryType(beeClients, config.namespace)

	if useDeploymentType {
		c.log.Infof("using deployment type %s", config.deploymentType)
	}

	return c.buildNodeClient(beeClients, config, discoveryType), nil
}

// nodeClientConfig holds the extracted configuration values
type nodeClientConfig struct {
	namespace      string
	clusterName    string
	labelSelector  string
	deploymentType string
	inCluster      bool
	nodeGroups     []string
}

// validateNodeClientInputs validates the input parameters for node client creation
func (c *command) validateNodeClientInputs(useDeploymentType bool) error {
	namespace := c.globalConfig.GetString(optionNameNamespace)
	clusterName := c.globalConfig.GetString(optionNameClusterName)

	if clusterName == "" && namespace == "" {
		return errors.New("either cluster name or namespace must be provided")
	}

	if c.globalConfig.IsSet(optionNameNamespace) && namespace == "" {
		return errors.New("namespace cannot be empty if set")
	}

	if namespace == "" && useDeploymentType && !isValidDeploymentType(c.globalConfig.GetString(optionNameDeploymentType)) {
		return errors.New("unsupported deployment type: must be 'beekeeper' or 'helm'")
	}

	// Note: Namespace will be available either from:
	// 1. Explicit configuration (optionNameNamespace)
	// 2. Cluster setup (cluster.Namespace())
	// This ensures all K8s operations (pods, statefulsets, services, ingress) have the required namespace

	return nil
}

// extractNodeClientConfig extracts configuration values from global config
func (c *command) extractNodeClientConfig() nodeClientConfig {
	return nodeClientConfig{
		namespace:      c.globalConfig.GetString(optionNameNamespace),
		clusterName:    c.globalConfig.GetString(optionNameClusterName),
		labelSelector:  c.globalConfig.GetString(optionNameLabelSelector),
		deploymentType: c.globalConfig.GetString(optionNameDeploymentType),
		inCluster:      c.globalConfig.GetBool(optionNameInCluster),
		nodeGroups:     c.globalConfig.GetStringSlice(optionNameNodeGroups),
	}
}

// setupBeeClients sets up bee clients if cluster is specified
func (c *command) setupBeeClients(ctx context.Context, clusterName string) (map[string]*bee.Client, string, error) {
	if clusterName == "" {
		return nil, "", nil
	}

	cluster, err := c.setupCluster(ctx, clusterName, false)
	if err != nil {
		return nil, "", fmt.Errorf("setting up cluster %s: %w", clusterName, err)
	}

	beeClients, err := cluster.NodesClients(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve node clients: %w", err)
	}

	return beeClients, cluster.Namespace(), nil
}

// determineDiscoveryType determines the discovery type based on available resources
func (c *command) determineDiscoveryType(beeClients map[string]*bee.Client, namespace string) node.DiscoveryType {
	if len(beeClients) > 0 {
		return node.DiscoveryTypeBeeClients
	}

	if namespace != "" {
		return node.DiscoveryTypeNamespace
	}

	return node.DiscoveryTypeNamespace
}

// buildNodeClient creates the node client with the provided configuration
func (c *command) buildNodeClient(beeClients map[string]*bee.Client, config nodeClientConfig, discoveryType node.DiscoveryType) *node.Client {
	return node.New(&node.ClientConfig{
		Log:            c.log,
		HTTPClient:     c.httpClient,
		K8sClient:      c.k8sClient,
		BeeClients:     beeClients,
		Namespace:      config.namespace,
		LabelSelector:  config.labelSelector,
		DeploymentType: node.DeploymentType(config.deploymentType),
		InCluster:      config.inCluster,
		DiscoveryType:  discoveryType,
		NodeGroups:     config.nodeGroups,
	})
}

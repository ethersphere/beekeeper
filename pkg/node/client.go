// Package node provides node discovery and management capabilities for the beekeeper system.
// It supports multiple discovery methods:
// 1. BeeClients-based discovery (using pre-configured orchestration clients)
// 2. Namespace-based discovery (using K8s services/ingress with label selectors)
// 3. StatefulSet-based discovery (discovering nodes from statefulset names and replica counts)
package node

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

type DeploymentType string

const (
	DeploymentTypeBeekeeper DeploymentType = "beekeeper"
	DeploymentTypeHelm      DeploymentType = "helm"
)

type DiscoveryType string

const (
	DiscoveryTypeBeeClients  DiscoveryType = "beeclients"
	DiscoveryTypeNamespace   DiscoveryType = "namespace"
	DiscoveryTypeStatefulSet DiscoveryType = "statefulset"
)

var ErrNodesNotFound = fmt.Errorf("nodes not found")

type NodeProvider interface {
	GetNodes(ctx context.Context) (NodeList, error)
	Namespace() string
}

type ClientConfig struct {
	Log            logging.Logger
	HTTPClient     *http.Client
	K8sClient      *k8s.Client
	BeeClients     orchestration.ClientMap
	Namespace      string
	LabelSelector  string
	DeploymentType DeploymentType
	InCluster      bool
	// Discovery method selection
	DiscoveryType DiscoveryType // Choose discovery method: beeclients, namespace, or statefulset
	NodeGroups    []string      // Node groups for filtering nodes (only used with beekeeper deployment)
	// StatefulSet-based discovery options
	StatefulSetNames []string // Names of statefulsets to discover nodes from
}

type Client struct {
	log            logging.Logger
	httpClient     *http.Client
	k8sClient      *k8s.Client
	beeClients     orchestration.ClientMap
	namespace      string
	labelSelector  string
	deploymentType DeploymentType
	inCluster      bool
	discoveryType  DiscoveryType
	nodeGroups     []string
	// StatefulSet-based discovery fields
	statefulSetNames []string
}

func New(cfg *ClientConfig) *Client {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}

	return &Client{
		log:              cfg.Log,
		httpClient:       cfg.HTTPClient,
		k8sClient:        cfg.K8sClient,
		beeClients:       cfg.BeeClients,
		namespace:        cfg.Namespace,
		labelSelector:    cfg.LabelSelector,
		deploymentType:   cfg.DeploymentType,
		inCluster:        cfg.InCluster,
		discoveryType:    cfg.DiscoveryType,
		nodeGroups:       cfg.NodeGroups,
		statefulSetNames: cfg.StatefulSetNames,
	}
}

func (sc *Client) Namespace() string {
	return sc.namespace
}

func (sc *Client) GetNodes(ctx context.Context) (nodes NodeList, err error) {
	defer func() {
		if err != nil || nodes == nil {
			return
		}
		for i := range nodes {
			sc.log.Debugf("adding node %s with endpoint %s", nodes[i].Name(), nodes[i].Client().Host())
		}
		sc.log.Infof("found %d nodes", len(nodes))
	}()

	// Choose discovery method based on configuration
	switch sc.discoveryType {
	case DiscoveryTypeStatefulSet:
		// Validate configuration before proceeding
		if err := sc.ValidateStatefulSetDiscovery(); err != nil {
			return nil, fmt.Errorf("statefulset discovery validation failed: %w", err)
		}
		return sc.getStatefulSetNodes(ctx)

	case DiscoveryTypeNamespace:
		if sc.namespace == "" {
			return nil, fmt.Errorf("namespace not provided for namespace discovery")
		}
		return sc.getNamespaceNodes(ctx)

	case DiscoveryTypeBeeClients:
		if len(sc.beeClients) == 0 {
			return nil, fmt.Errorf("bee clients not provided")
		}
		filteredClients := sc.beeClients.FilterByNodeGroups(sc.nodeGroups)
		if len(filteredClients) == 0 {
			return nil, ErrNodesNotFound
		}
		nodes = make(NodeList, 0, len(filteredClients))
		for _, beeClient := range filteredClients {
			nodes = append(nodes, *NewNode(beeClient.API(), sc.nodeName(beeClient.Name())))
		}
		return nodes.Sort(), nil

	default:
		return nil, fmt.Errorf("unknown discovery type: %s", sc.discoveryType)
	}
}

func (sc *Client) getNamespaceNodes(ctx context.Context) (nodes []Node, err error) {
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

func (sc *Client) getServiceNodes(ctx context.Context) ([]Node, error) {
	svcNodes, err := sc.k8sClient.Service.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list api services: %w", err)
	}

	nodes := make([]Node, len(svcNodes))
	for i, node := range svcNodes {
		parsedURL, err := url.Parse(node.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("extract base URL: %w", err)
		}

		apiClient, err := api.NewClient(parsedURL, sc.httpClient)
		if err != nil {
			return nil, fmt.Errorf("create api client: %w", err)
		}

		nodes[i] = *NewNode(apiClient, sc.nodeName(node.Name))
	}

	return nodes, nil
}

func (sc *Client) getIngressNodes(ctx context.Context) ([]Node, error) {
	ingressNodes, err := sc.k8sClient.Ingress.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list ingress api nodes hosts: %w", err)
	}

	ingressRouteNodes, err := sc.k8sClient.IngressRoute.GetNodes(ctx, sc.namespace, sc.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("list ingress route api nodes hosts: %w", err)
	}

	allNodes := append(ingressNodes, ingressRouteNodes...)
	nodes := make([]Node, len(allNodes))
	for i, node := range allNodes {
		apiURL, err := url.Parse(fmt.Sprintf("http://%s", node.Host))
		if err != nil {
			return nil, fmt.Errorf("extract base URL: %w", err)
		}

		apiClient, err := api.NewClient(apiURL, sc.httpClient)
		if err != nil {
			return nil, fmt.Errorf("create api client: %w", err)
		}

		nodes[i] = *NewNode(apiClient, sc.nodeName(node.Name))
	}

	return nodes, nil
}

// nodeName returns the name of the node, and adds suffix based on deployment type.
// In case of Beekeeper deployment, it adds "-0" suffix, because there is only one replica.
// In case of other deployments, it returns the name as is.
func (sc *Client) nodeName(name string) string {
	if sc.deploymentType == DeploymentTypeBeekeeper {
		return fmt.Sprintf("%s-0", name)
	}
	return name
}

// getStatefulSetNodes discovers nodes based on statefulset names and their replicas
func (sc *Client) getStatefulSetNodes(ctx context.Context) ([]Node, error) {
	if sc.k8sClient == nil {
		return nil, fmt.Errorf("k8s client not provided")
	}

	if sc.namespace == "" {
		return nil, fmt.Errorf("namespace not provided")
	}

	if len(sc.statefulSetNames) == 0 {
		return nil, fmt.Errorf("no statefulset names provided")
	}

	sc.log.Infof("discovering nodes from %d statefulsets: %v", len(sc.statefulSetNames), sc.statefulSetNames)

	var allNodes []Node
	for _, ssName := range sc.statefulSetNames {
		nodes, err := sc.getNodesFromStatefulSet(ctx, ssName)
		if err != nil {
			sc.log.Warningf("failed to get nodes from statefulset %s: %v", ssName, err)
			continue
		}
		allNodes = append(allNodes, nodes...)
	}

	if len(allNodes) == 0 {
		return nil, ErrNodesNotFound
	}

	return allNodes, nil
}

// getNodesFromStatefulSet discovers all nodes for a specific statefulset
func (sc *Client) getNodesFromStatefulSet(ctx context.Context, statefulSetName string) ([]Node, error) {
	// Get the statefulset to determine replica count
	ss, err := sc.k8sClient.StatefulSet.Get(ctx, statefulSetName, sc.namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulset %s: %w", statefulSetName, err)
	}

	replicas := int32(1)
	if ss.Spec.Replicas != nil {
		replicas = *ss.Spec.Replicas
	}

	sc.log.Debugf("statefulset %s has %d replicas", statefulSetName, replicas)

	var nodes []Node
	for i := int32(0); i < replicas; i++ {
		podName := fmt.Sprintf("%s-%d", statefulSetName, i)
		node, err := sc.createNodeFromPod(ctx, podName, statefulSetName, int(i))
		if err != nil {
			sc.log.Warningf("failed to create node for pod %s: %v", podName, err)
			continue
		}
		nodes = append(nodes, *node)
	}

	return nodes, nil
}

// createNodeFromPod creates a Node from a pod, determining the endpoint via service or direct pod access
func (sc *Client) createNodeFromPod(ctx context.Context, podName, statefulSetName string, replicaIndex int) (*Node, error) {
	// Try to get the pod to verify it exists
	pod, err := sc.k8sClient.Pods.Get(ctx, podName, sc.namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	// Determine the endpoint
	var endpoint string
	var nodeName string

	if sc.inCluster {
		// In-cluster: use service discovery
		svcNodes, err := sc.k8sClient.Service.GetNodes(ctx, sc.namespace, fmt.Sprintf("app.kubernetes.io/instance=%s", statefulSetName))
		if err == nil && len(svcNodes) > 0 {
			// Use service endpoint
			endpoint = svcNodes[0].Endpoint
			nodeName = sc.nodeNameFromStatefulSet(statefulSetName, replicaIndex)
		} else {
			// Fallback to pod IP
			if pod.Status.PodIP == "" {
				return nil, fmt.Errorf("pod %s has no IP address", podName)
			}
			endpoint = fmt.Sprintf("http://%s:1633", pod.Status.PodIP) // Assuming bee API port
			nodeName = sc.nodeNameFromStatefulSet(statefulSetName, replicaIndex)
		}
	} else {
		// External access: try ingress first, then pod IP
		ingressNodes, err := sc.k8sClient.Ingress.GetNodes(ctx, sc.namespace, fmt.Sprintf("app.kubernetes.io/instance=%s", statefulSetName))
		if err == nil && len(ingressNodes) > 0 {
			endpoint = fmt.Sprintf("http://%s", ingressNodes[0].Host)
			nodeName = sc.nodeNameFromStatefulSet(statefulSetName, replicaIndex)
		} else {
			// Fallback to pod IP
			if pod.Status.PodIP == "" {
				return nil, fmt.Errorf("pod %s has no IP address", podName)
			}
			endpoint = fmt.Sprintf("http://%s:1633", pod.Status.PodIP)
			nodeName = sc.nodeNameFromStatefulSet(statefulSetName, replicaIndex)
		}
	}

	// Parse endpoint and create API client
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint %s: %w", endpoint, err)
	}

	apiClient, err := api.NewClient(parsedURL, sc.httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client for %s: %w", endpoint, err)
	}

	return NewNode(apiClient, nodeName), nil
}

// nodeNameFromStatefulSet creates a node name from statefulset name and replica index
func (sc *Client) nodeNameFromStatefulSet(statefulSetName string, replicaIndex int) string {
	return fmt.Sprintf("%s-%d", statefulSetName, replicaIndex)
}

// GetNodesByStatefulSets is a convenience method to get nodes from specific statefulsets
func (sc *Client) GetNodesByStatefulSets(ctx context.Context, statefulSetNames []string) (NodeList, error) {
	if sc.k8sClient == nil {
		return nil, fmt.Errorf("k8s client not provided")
	}

	if sc.namespace == "" {
		return nil, fmt.Errorf("namespace not provided")
	}

	if len(statefulSetNames) == 0 {
		return nil, fmt.Errorf("no statefulset names provided")
	}

	sc.log.Infof("discovering nodes from statefulsets: %v", statefulSetNames)

	var allNodes []Node
	for _, ssName := range statefulSetNames {
		nodes, err := sc.getNodesFromStatefulSet(ctx, ssName)
		if err != nil {
			sc.log.Warningf("failed to get nodes from statefulset %s: %v", ssName, err)
			continue
		}
		allNodes = append(allNodes, nodes...)
	}

	if len(allNodes) == 0 {
		return nil, ErrNodesNotFound
	}

	return NodeList(allNodes).Sort(), nil
}

// ValidateStatefulSetDiscovery validates the configuration for statefulset discovery
func (sc *Client) ValidateStatefulSetDiscovery() error {
	if sc.discoveryType != DiscoveryTypeStatefulSet {
		return nil // Not using statefulset discovery, no validation needed
	}

	if sc.k8sClient == nil {
		return fmt.Errorf("k8s client is required for statefulset discovery")
	}

	if sc.namespace == "" {
		return fmt.Errorf("namespace is required for statefulset discovery")
	}

	if len(sc.statefulSetNames) == 0 {
		return fmt.Errorf("statefulset names must be provided for statefulset discovery")
	}

	return nil
}

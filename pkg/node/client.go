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
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
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

const (
	instanceLabel  = "app.kubernetes.io/instance"
	apiPortName    = "api"
	defaultAPIPort = 1633
)

var ErrNodesNotFound = fmt.Errorf("nodes not found")

type NodeProvider interface {
	GetNodes(ctx context.Context) (NodeList, error)
	Namespace() string
}

type ClientConfig struct {
	Log              logging.Logger
	HTTPClient       *http.Client
	K8sClient        *k8s.Client
	BeeClients       orchestration.ClientMap
	Namespace        string
	LabelSelector    string
	DeploymentType   DeploymentType
	InCluster        bool
	DiscoveryType    DiscoveryType // Discovery method: beeclients, namespace, or statefulset
	NodeGroups       []string      // Node groups for filtering nodes (only used with beeclients discovery)
	StatefulSetNames []string      // Names of statefulsets to discover nodes from (only used with statefulset discovery)
}

type Client struct {
	log              logging.Logger
	httpClient       *http.Client
	k8sClient        *k8s.Client
	beeClients       orchestration.ClientMap
	namespace        string
	labelSelector    string
	deploymentType   DeploymentType
	inCluster        bool
	discoveryType    DiscoveryType
	nodeGroups       []string
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

	switch sc.discoveryType {
	case DiscoveryTypeStatefulSet:
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
		return nil, fmt.Errorf("unknown discovery type: %q", sc.discoveryType)
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

// getStatefulSetNodes discovers nodes from the pods of the configured
// statefulsets. A statefulset or a pod that can not be resolved is logged and
// skipped, the discovery fails only if no node is found at all.
func (sc *Client) getStatefulSetNodes(ctx context.Context) (NodeList, error) {
	if sc.k8sClient == nil {
		return nil, fmt.Errorf("k8s client not provided")
	}

	if sc.namespace == "" {
		return nil, fmt.Errorf("namespace not provided")
	}

	if len(sc.statefulSetNames) == 0 {
		return nil, fmt.Errorf("statefulset names not provided")
	}

	sc.log.Infof("discovering nodes from %d statefulsets: %v", len(sc.statefulSetNames), sc.statefulSetNames)

	var nodes NodeList
	seen := make(map[string]struct{}, len(sc.statefulSetNames))
	for _, name := range sc.statefulSetNames {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		ssNodes, err := sc.getNodesFromStatefulSet(ctx, name)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			sc.log.Warningf("skipping statefulset %s: %v", name, err)
			continue
		}
		nodes = append(nodes, ssNodes...)
	}

	if len(nodes) == 0 {
		return nil, ErrNodesNotFound
	}

	return nodes.Sort(), nil
}

// getNodesFromStatefulSet returns a node for every replica of the statefulset.
func (sc *Client) getNodesFromStatefulSet(ctx context.Context, name string) ([]Node, error) {
	ss, err := sc.k8sClient.StatefulSet.Get(ctx, name, sc.namespace)
	if err != nil {
		return nil, fmt.Errorf("get statefulset: %w", err)
	}

	replicas := 1
	if ss.Spec.Replicas != nil {
		replicas = int(*ss.Spec.Replicas)
	}

	sc.log.Debugf("statefulset %s has %d replicas", name, replicas)

	endpoints := sc.statefulSetEndpoints(ctx, ss)

	nodes := make([]Node, 0, replicas)
	for i := range replicas {
		podName := fmt.Sprintf("%s-%d", name, i)

		pod, err := sc.k8sClient.Pods.Get(ctx, podName, sc.namespace)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			sc.log.Warningf("skipping pod %s: %v", podName, err)
			continue
		}

		endpoint, err := sc.podEndpoint(pod, name, replicas, endpoints)
		if err != nil {
			sc.log.Warningf("skipping pod %s: %v", podName, err)
			continue
		}

		apiURL, err := url.Parse(endpoint)
		if err != nil {
			sc.log.Warningf("skipping pod %s: parse endpoint %s: %v", podName, endpoint, err)
			continue
		}

		apiClient, err := api.NewClient(apiURL, sc.httpClient)
		if err != nil {
			sc.log.Warningf("skipping pod %s: create api client: %v", podName, err)
			continue
		}

		nodes = append(nodes, *NewNode(apiClient, podName))
	}

	return nodes, nil
}

// statefulSetEndpoints returns API endpoints of the resources that belong to
// the same instance as the statefulset, keyed by the resource name. Services
// are used inside of the cluster, Ingresses and IngressRoutes outside of it.
func (sc *Client) statefulSetEndpoints(ctx context.Context, ss *appsv1.StatefulSet) map[string]string {
	instance := ss.Labels[instanceLabel]
	if instance == "" {
		instance = ss.Name
	}
	selector := fmt.Sprintf("%s=%s", instanceLabel, instance)

	endpoints := make(map[string]string)

	if sc.inCluster {
		svcNodes, err := sc.k8sClient.Service.GetNodes(ctx, sc.namespace, selector)
		if err != nil {
			sc.log.Warningf("statefulset %s: list api services: %v", ss.Name, err)
		}
		for _, n := range svcNodes {
			endpoints[n.Name] = n.Endpoint
		}
		return endpoints
	}

	ingressNodes, err := sc.k8sClient.Ingress.GetNodes(ctx, sc.namespace, selector)
	if err != nil {
		sc.log.Warningf("statefulset %s: list ingress api nodes hosts: %v", ss.Name, err)
	}

	ingressRouteNodes, err := sc.k8sClient.IngressRoute.GetNodes(ctx, sc.namespace, selector)
	if err != nil {
		sc.log.Warningf("statefulset %s: list ingress route api nodes hosts: %v", ss.Name, err)
	}

	for _, n := range append(ingressNodes, ingressRouteNodes...) {
		endpoints[n.Name] = fmt.Sprintf("http://%s", n.Host)
	}

	return endpoints
}

// podEndpoint returns the API endpoint of the pod. A resource named after the
// pod is used first (Helm deployment), then a resource named after a single
// replica statefulset (Beekeeper deployment), and the pod IP otherwise.
func (sc *Client) podEndpoint(pod *v1.Pod, statefulSetName string, replicas int, endpoints map[string]string) (string, error) {
	if endpoint, ok := endpoints[pod.Name]; ok {
		return endpoint, nil
	}

	if endpoint, ok := endpoints[statefulSetName]; ok && replicas == 1 {
		return endpoint, nil
	}

	if pod.Status.PodIP == "" {
		return "", fmt.Errorf("no api service or ingress found and pod has no IP address")
	}

	if !sc.inCluster {
		sc.log.Warningf("no ingress found for pod %s, using pod IP %s which may not be reachable from outside of the cluster", pod.Name, pod.Status.PodIP)
	}

	return fmt.Sprintf("http://%s", net.JoinHostPort(pod.Status.PodIP, strconv.Itoa(podAPIPort(pod)))), nil
}

// podAPIPort returns the container port named api, or the default Bee API port.
func podAPIPort(pod *v1.Pod) int {
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.Name == apiPortName {
				return int(port.ContainerPort)
			}
		}
	}
	return defaultAPIPort
}

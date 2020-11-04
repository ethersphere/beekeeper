package bee

import (
	"context"

	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

// NodeGroup ...
type NodeGroup struct {
	Name string

	nodes map[string]*Client
	opts  NodeGroupOptions
	// set when added to cluster
	cluster *DynamicCluster
	k8s     *k8sBee.Client
}

// NodeGroupOptions ...
type NodeGroupOptions struct {
	Annotations               map[string]string
	ClefImage                 string
	ClefImagePullPolicy       string
	Image                     string
	ImagePullPolicy           string
	IngressAnnotations        map[string]string
	IngressDebugAnnotations   map[string]string
	Labels                    map[string]string
	LimitCPU                  string
	LimitMemory               string
	NodeSelector              map[string]string
	PersistenceEnabled        bool
	PersistenceStorageClass   string
	PersistanceStorageRequest string
	PodManagementPolicy       string
	RestartPolicy             string
	RequestCPU                string
	RequestMemory             string
	UpdateStrategy            string
}

// NewNodeGroup ...
func NewNodeGroup(name string, o NodeGroupOptions) *NodeGroup {
	return &NodeGroup{
		Name:  name,
		nodes: make(map[string]*Client),
		opts:  o,
	}
}

// Nodes returns map of node groups in the cluster
func (g *NodeGroup) Nodes() (l map[string]*Client) {
	return g.nodes
}

// Node returns node's client
func (g *NodeGroup) Node(name string) *Client {
	return g.nodes[name]
}

// NodeStartOptions ...
type NodeStartOptions struct {
	Name         string
	Config       k8sBee.Config
	ClefKey      string
	ClefPassword string
	LibP2PKey    string
	SwarmKey     string
}

// NodeStart ...
func (g *NodeGroup) NodeStart(ctx context.Context, o NodeStartOptions) (err error) {
	aURL, err := g.cluster.apiURL(o.Name)
	if err != nil {
		return err
	}

	dURL, err := g.cluster.debugAPIURL(o.Name)
	if err != nil {
		return err
	}

	c := NewClient(ClientOptions{
		APIURL:              aURL,
		APIInsecureTLS:      g.cluster.apiInsecureTLS,
		DebugAPIURL:         dURL,
		DebugAPIInsecureTLS: g.cluster.debugAPIInsecureTLS,
	})
	g.nodes[o.Name] = &c

	labels := mergeMaps(g.opts.Labels, map[string]string{
		"app.kubernetes.io/instance": o.Name,
	})

	return g.k8s.NodeStart(ctx, k8sBee.NodeStartOptions{
		// Bee configuration
		Config: o.Config,
		// Kubernetes configuration
		Name:                      o.Name,
		Namespace:                 g.cluster.namespace,
		Annotations:               g.opts.Annotations,
		ClefImage:                 g.opts.ClefImage,
		ClefImagePullPolicy:       g.opts.ClefImagePullPolicy,
		ClefKey:                   o.ClefKey,
		ClefPassword:              o.ClefPassword,
		Image:                     g.opts.Image,
		ImagePullPolicy:           g.opts.ImagePullPolicy,
		IngressAnnotations:        g.opts.IngressAnnotations,
		IngressHost:               g.cluster.ingressHost(o.Name),
		IngressDebugAnnotations:   g.opts.IngressDebugAnnotations,
		IngressDebugHost:          g.cluster.ingressDebugHost(o.Name),
		Labels:                    labels,
		LibP2PKey:                 o.LibP2PKey,
		LimitCPU:                  g.opts.LimitCPU,
		LimitMemory:               g.opts.LimitMemory,
		NodeSelector:              g.opts.NodeSelector,
		PersistenceEnabled:        g.opts.PersistenceEnabled,
		PersistenceStorageClass:   g.opts.PersistenceStorageClass,
		PersistanceStorageRequest: g.opts.PersistanceStorageRequest,
		PodManagementPolicy:       g.opts.PodManagementPolicy,
		RestartPolicy:             g.opts.RestartPolicy,
		RequestCPU:                g.opts.RequestCPU,
		RequestMemory:             g.opts.RequestMemory,
		Selector:                  labels,
		SwarmKey:                  o.SwarmKey,
		UpdateStrategy:            g.opts.UpdateStrategy,
	})
}

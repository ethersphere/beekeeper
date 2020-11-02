package bee

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

const (
	labelBy        = "beekeeper"
	labelComponent = "bee"
	labelName      = "bee"
)

// DynamicCluster ...
type DynamicCluster struct {
	Name string

	namespace   string
	annotations map[string]string
	labels      map[string]string
	nodeGroups  map[string]NodeGroup

	beeK8S *k8sBee.Client
	k8s    *k8s.Client
}

// DynamicClusterOptions ...
type DynamicClusterOptions struct {
	Name           string
	Namespace      string
	KubeconfigPath string
}

// NewDynamicCluster ...
func NewDynamicCluster(o DynamicClusterOptions) *DynamicCluster {
	k8s := k8s.NewClient(&k8s.ClientOptions{KubeconfigPath: o.KubeconfigPath})

	return &DynamicCluster{
		Name: o.Name,

		namespace: o.Namespace,
		annotations: map[string]string{
			"created-by":        labelBy,
			"beekeeper/version": beekeeper.Version,
		},
		labels: map[string]string{
			"app.kubernetes.io/managed-by": labelBy,
			"app.kubernetes.io/name":       labelName,
		},
		nodeGroups: make(map[string]NodeGroup),

		k8s:    k8s,
		beeK8S: k8sBee.NewClient(k8s),
	}
}

// NodeGroups returns map of node groups in the cluster
func (dc *DynamicCluster) NodeGroups() (l map[string]NodeGroup) {
	return dc.nodeGroups
}

// NodeGroup returns node group
func (dc *DynamicCluster) NodeGroup(name string) NodeGroup {
	return dc.nodeGroups[name]
}

// Start starts cluster with given options
// func (dc *DynamicCluster) Start(ctx context.Context) (err error) {
// check if namespace exists, and create one if doesn't
// return
// }

// NodeGroup ...
type NodeGroup struct {
	Name    string
	Options NodeGroupOptions

	namespace   string
	annotations map[string]string
	labels      map[string]string
	nodes       map[string]Client
	version     string
}

// NodeGroupOptions ...
type NodeGroupOptions struct {
	ClefImage                 string
	ClefImagePullPolicy       string
	Image                     string
	ImagePullPolicy           string
	IngressAnnotations        map[string]string
	IngressDebugAnnotations   map[string]string
	InsecureTLS               bool
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
func (dc *DynamicCluster) NewNodeGroup(name string, o NodeGroupOptions) {
	version := strings.Split(o.Image, ":")[1]
	dc.nodeGroups[name] = NodeGroup{
		Name:    name,
		Options: o,

		namespace:   dc.namespace,
		annotations: dc.annotations,
		labels: mergeMaps(dc.labels, map[string]string{
			"app.kubernetes.io/component": labelComponent,
			"app.kubernetes.io/part-of":   name,
			"app.kubernetes.io/version":   version,
		}),
		nodes:   make(map[string]Client),
		version: version,
	}
}

// Nodes returns map of node groups in the cluster
func (ng NodeGroup) Nodes() (l map[string]Client) {
	return ng.nodes
}

// Node returns node's client
func (ng NodeGroup) Node(name string) Client {
	return ng.nodes[name]
}

// NodeStartOptions ...
type NodeStartOptions struct {
	Name             string
	Config           k8sBee.Config
	Annotations      map[string]string
	APIURL           string
	ClefKey          string
	ClefPassword     string
	DebugAPIURL      string
	IngressHost      string
	IngressDebugHost string
	Labels           map[string]string
	LibP2PKey        string
	SwarmKey         string
}

// NodeStart ...
func (dc *DynamicCluster) NodeStart(ctx context.Context, groupName string, o NodeStartOptions) (err error) {
	apiURL, err := url.Parse(o.APIURL)
	if err != nil {
		return fmt.Errorf("bad API url (group: %s, node: %s): %s", groupName, o.Name, err)
	}
	debugAPIURL, err := url.Parse(o.DebugAPIURL)
	if err != nil {
		return fmt.Errorf("bad debug API url (group: %s, node: %s): %s", groupName, o.Name, err)
	}

	g := dc.nodeGroups[groupName]
	g.nodes[o.Name] = NewClient(ClientOptions{
		APIURL:              apiURL,
		APIInsecureTLS:      g.Options.InsecureTLS,
		DebugAPIURL:         debugAPIURL,
		DebugAPIInsecureTLS: g.Options.InsecureTLS,
	})

	labels := mergeMaps(g.labels, map[string]string{
		"app.kubernetes.io/instance": o.Name,
	})

	return dc.beeK8S.NodeStart(ctx, k8sBee.NodeStartOptions{
		// Bee configuration
		Config: o.Config,
		// Kubernetes configuration
		Name:                      o.Name,
		Namespace:                 g.namespace,
		Annotations:               mergeMaps(g.annotations, o.Annotations),
		ClefImage:                 g.Options.ClefImage,
		ClefImagePullPolicy:       g.Options.ClefImagePullPolicy,
		ClefKey:                   o.ClefKey,
		ClefPassword:              o.ClefPassword,
		Image:                     g.Options.Image,
		ImagePullPolicy:           g.Options.ImagePullPolicy,
		IngressAnnotations:        g.Options.IngressAnnotations,
		IngressHost:               o.IngressHost,
		IngressDebugAnnotations:   g.Options.IngressDebugAnnotations,
		IngressDebugHost:          o.IngressDebugHost,
		Labels:                    mergeMaps(labels, o.Labels),
		LibP2PKey:                 o.LibP2PKey,
		LimitCPU:                  g.Options.LimitCPU,
		LimitMemory:               g.Options.LimitMemory,
		NodeSelector:              g.Options.NodeSelector,
		PersistenceEnabled:        g.Options.PersistenceEnabled,
		PersistenceStorageClass:   g.Options.PersistenceStorageClass,
		PersistanceStorageRequest: g.Options.PersistanceStorageRequest,
		PodManagementPolicy:       g.Options.PodManagementPolicy,
		RestartPolicy:             g.Options.RestartPolicy,
		RequestCPU:                g.Options.RequestCPU,
		RequestMemory:             g.Options.RequestMemory,
		Selector: map[string]string{
			"app.kubernetes.io/component":  labelComponent,
			"app.kubernetes.io/instance":   o.Name,
			"app.kubernetes.io/managed-by": labelBy,
			"app.kubernetes.io/name":       labelName,
			"app.kubernetes.io/part-of":    g.Name,
			"app.kubernetes.io/version":    g.version,
		},
		SwarmKey:       o.SwarmKey,
		UpdateStrategy: g.Options.UpdateStrategy,
	})
}

func mergeMaps(a, b map[string]string) map[string]string {
	m := map[string]string{}
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}

	return m
}

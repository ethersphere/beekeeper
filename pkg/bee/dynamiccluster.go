package bee

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

// DynamicCluster ...
type DynamicCluster struct {
	Name string

	beeK8S     *k8sBee.Client
	k8s        *k8s.Client
	nodeGroups map[string]NodeGroup
}

// DynamicClusterOptions ...
type DynamicClusterOptions struct {
	Name           string
	KubeconfigPath string
}

// NewDynamicCluster ...
func NewDynamicCluster(o DynamicClusterOptions) *DynamicCluster {
	k8s := k8s.NewClient(&k8s.ClientOptions{KubeconfigPath: o.KubeconfigPath})

	return &DynamicCluster{
		Name:       o.Name,
		k8s:        k8s,
		beeK8S:     k8sBee.NewClient(k8s),
		nodeGroups: make(map[string]NodeGroup),
	}
}

// Start starts cluster with given options
func (dc *DynamicCluster) Start(ctx context.Context) (err error) {
	return
}

// NodeGroup ...
type NodeGroup struct {
	nodes   map[string]Client
	Options NodeGroupOptions
}

// NodeGroupOptions ...
type NodeGroupOptions struct {
	Name                      string
	Namespace                 string
	Annotations               map[string]string
	ClefImage                 string
	ClefImagePullPolicy       string
	ClefKey                   string
	ClefPassword              string
	Labels                    map[string]string
	LimitCPU                  string
	LimitMemory               string
	Image                     string
	ImagePullPolicy           string
	IngressAnnotations        map[string]string
	IngressHost               string
	IngressDebugAnnotations   map[string]string
	IngressDebugHost          string
	LibP2PKey                 string
	NodeSelector              map[string]string
	PersistenceEnabled        bool
	PersistenceStorageClass   string
	PersistanceStorageRequest string
	PodManagementPolicy       string
	RestartPolicy             string
	RequestCPU                string
	RequestMemory             string
	Selector                  map[string]string
	SwarmKey                  string
	UpdateStrategy            string
}

// NewNodeGroup ...
func (dc *DynamicCluster) NewNodeGroup(o NodeGroupOptions) {
	dc.nodeGroups[o.Name] = NodeGroup{
		nodes:   make(map[string]Client),
		Options: o,
	}
}

// NodeStartOptions ...
type NodeStartOptions struct {
	Name         string
	Config       k8sBee.Config
	GroupName    string
	GroupOptions NodeGroupOptions
}

// NodeStart ...
func (dc *DynamicCluster) NodeStart(ctx context.Context, o NodeStartOptions) (err error) {
	return dc.beeK8S.NodeStart(ctx, k8sBee.NodeStartOptions{
		// Bee configuration
		Config: o.Config,
		// Kubernetes configuration
		Name:                      o.Name,
		Namespace:                 o.GroupOptions.Namespace,
		Annotations:               o.GroupOptions.Annotations,
		ClefImage:                 o.GroupOptions.ClefImage,
		ClefImagePullPolicy:       o.GroupOptions.ClefImagePullPolicy,
		ClefKey:                   o.GroupOptions.ClefKey,
		ClefPassword:              o.GroupOptions.ClefKey,
		Labels:                    o.GroupOptions.Labels,
		LimitCPU:                  o.GroupOptions.LimitCPU,
		LimitMemory:               o.GroupOptions.LimitMemory,
		Image:                     o.GroupOptions.Image,
		ImagePullPolicy:           o.GroupOptions.ImagePullPolicy,
		IngressAnnotations:        o.GroupOptions.IngressAnnotations,
		IngressHost:               o.GroupOptions.IngressHost,
		IngressDebugAnnotations:   o.GroupOptions.IngressDebugAnnotations,
		IngressDebugHost:          o.GroupOptions.IngressDebugHost,
		LibP2PKey:                 o.GroupOptions.LibP2PKey,
		NodeSelector:              o.GroupOptions.NodeSelector,
		PersistenceEnabled:        o.GroupOptions.PersistenceEnabled,
		PersistenceStorageClass:   o.GroupOptions.PersistenceStorageClass,
		PersistanceStorageRequest: o.GroupOptions.PersistanceStorageRequest,
		PodManagementPolicy:       o.GroupOptions.PodManagementPolicy,
		RestartPolicy:             o.GroupOptions.RestartPolicy,
		RequestCPU:                o.GroupOptions.RequestCPU,
		RequestMemory:             o.GroupOptions.RequestMemory,
		Selector:                  o.GroupOptions.Selector,
		SwarmKey:                  o.GroupOptions.SwarmKey,
		UpdateStrategy:            o.GroupOptions.UpdateStrategy,
	})
}

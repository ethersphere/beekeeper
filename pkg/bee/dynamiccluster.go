package bee

import (
	"fmt"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	k8sBee "github.com/ethersphere/beekeeper/pkg/k8s/bee"
)

// DynamicCluster ...
type DynamicCluster struct {
	Name string

	annotations         map[string]string
	apiDomain           string
	apiInsecureTLS      bool
	apiScheme           string
	debugAPIDomain      string
	debugAPIInsecureTLS bool
	debugAPIScheme      string
	k8s                 *k8s.Client
	labels              map[string]string
	namespace           string

	nodeGroups map[string]*NodeGroup
}

// DynamicClusterOptions ...
type DynamicClusterOptions struct {
	Annotations         map[string]string
	APIDomain           string
	APIInsecureTLS      bool
	APIScheme           string
	DebugAPIDomain      string
	DebugAPIInsecureTLS bool
	DebugAPIScheme      string
	KubeconfigPath      string
	Labels              map[string]string
	Namespace           string
}

// NewDynamicCluster ...
func NewDynamicCluster(name string, o DynamicClusterOptions) *DynamicCluster {
	k8s := k8s.NewClient(&k8s.ClientOptions{KubeconfigPath: o.KubeconfigPath})

	return &DynamicCluster{
		Name: name,

		annotations:         o.Annotations,
		apiDomain:           o.APIDomain,
		apiInsecureTLS:      o.APIInsecureTLS,
		apiScheme:           o.APIScheme,
		debugAPIDomain:      o.DebugAPIDomain,
		debugAPIInsecureTLS: o.DebugAPIInsecureTLS,
		debugAPIScheme:      o.DebugAPIScheme,
		k8s:                 k8s,
		labels:              o.Labels,
		namespace:           o.Namespace,

		nodeGroups: make(map[string]*NodeGroup),
	}
}

// AddNodeGroup ...
func (dc *DynamicCluster) AddNodeGroup(name string, o NodeGroupOptions) {
	g := NewNodeGroup(name, o)
	g.cluster = dc
	g.k8s = k8sBee.NewClient(g.cluster.k8s)
	g.opts.Annotations = mergeMaps(g.cluster.annotations, o.Annotations)
	g.opts.Labels = mergeMaps(g.cluster.labels, o.Labels)

	dc.nodeGroups[name] = g
}

// NodeGroups returns map of node groups in the cluster
func (dc *DynamicCluster) NodeGroups() (l map[string]*NodeGroup) {
	return dc.nodeGroups
}

// NodeGroup returns node group
func (dc *DynamicCluster) NodeGroup(name string) *NodeGroup {
	return dc.nodeGroups[name]
}

// apiURL generates URL for node's API
func (dc *DynamicCluster) apiURL(name string) (u *url.URL, err error) {
	u, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", dc.apiScheme, name, dc.namespace, dc.apiDomain))
	if err != nil {
		return nil, fmt.Errorf("bad API url for node %s: %s", name, err)
	}
	return
}

// ingressHost generates host for node's API ingress
func (dc *DynamicCluster) ingressHost(name string) string {
	return fmt.Sprintf("%s.%s.%s", name, dc.namespace, dc.apiDomain)
}

// debugAPIURL generates URL for node's DebugAPI
func (dc *DynamicCluster) debugAPIURL(name string) (u *url.URL, err error) {
	u, err = url.Parse(fmt.Sprintf("%s://%s-debug.%s.%s", dc.debugAPIScheme, name, dc.namespace, dc.debugAPIDomain))
	if err != nil {
		return nil, fmt.Errorf("bad debug API url for node %s: %s", name, err)
	}
	return
}

// ingressHost generates host for node's DebugAPI ingress
func (dc *DynamicCluster) ingressDebugHost(name string) string {
	return fmt.Sprintf("%s-debug.%s.%s", name, dc.namespace, dc.debugAPIDomain)
}

// mergeMaps joins two maps
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

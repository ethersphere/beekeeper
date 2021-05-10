package config

import (
	"reflect"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Cluster represents cluster configuration
type Cluster struct {
	// parent to inherit settings from
	*Inherit `yaml:",inline"`
	// Cluster configuration
	Name                *string                      `yaml:"name"`
	Namespace           *string                      `yaml:"namespace"`
	DisableNamespace    *bool                        `yaml:"disable-namespace"`
	APIDomain           *string                      `yaml:"api-domain"`
	APIInsecureTLS      *bool                        `yaml:"api-insecure-tls"`
	APIScheme           *string                      `yaml:"api-scheme"`
	DebugAPIDomain      *string                      `yaml:"debug-api-domain"`
	DebugAPIInsecureTLS *bool                        `yaml:"debug-api-insecure-tls"`
	DebugAPIScheme      *string                      `yaml:"debug-api-scheme"`
	NodeGroups          *map[string]ClusterNodeGroup `yaml:"node-groups"`
}

// ClusterNodeGroup represents node group in the cluster
type ClusterNodeGroup struct {
	Mode      string        `yaml:"mode"`
	BeeConfig string        `yaml:"bee-config"`
	Config    string        `yaml:"config"`
	Count     int           `yaml:"count"`
	Nodes     []ClusterNode `yaml:"nodes"`
}

// ClusterNode represents node in the cluster
type ClusterNode struct {
	Name         string `yaml:"name"`
	Bootnodes    string `yaml:"bootnodes"`
	ClefKey      string `yaml:"clef-key"`
	ClefPassword string `yaml:"clef-password"`
	LibP2PKey    string `yaml:"libp2p-key"`
	SwarmKey     string `yaml:"swarm-key"`
}

// Export exports Cluster to bee.ClusterOptions, skipping all other extra fields
func (c *Cluster) Export() (o bee.ClusterOptions) {
	localVal := reflect.ValueOf(c).Elem()
	localType := reflect.TypeOf(c).Elem()
	remoteVal := reflect.ValueOf(&o).Elem()

	for i := 0; i < localVal.NumField(); i++ {
		localField := localVal.Field(i)
		if localField.IsValid() && !localField.IsNil() {
			localFieldVal := localVal.Field(i).Elem()
			localFieldName := localType.Field(i).Name

			remoteFieldVal := remoteVal.FieldByName(localFieldName)
			if remoteFieldVal.IsValid() && remoteFieldVal.Type() == localFieldVal.Type() {
				remoteFieldVal.Set(localFieldVal)
			}
		}
	}

	return remoteVal.Interface().(bee.ClusterOptions)
}

// GetName returns cluster name
func (c *Cluster) GetName() string {
	if c.Name == nil {
		return "noname"
	}
	return *c.Name
}

// GetNamespace returns cluster namespace
func (c *Cluster) GetNamespace() string {
	if c.Name == nil {
		return "nonamespace"
	}
	return *c.Namespace
}

// GetNodeGroups returns cluster node groups
func (c *Cluster) GetNodeGroups() map[string]ClusterNodeGroup {
	if c.NodeGroups == nil {
		return nil
	}
	return *c.NodeGroups
}

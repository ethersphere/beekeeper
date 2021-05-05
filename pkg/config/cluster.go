package config

import (
	"reflect"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

type Cluster struct {
	// parrent to inherit settings from
	*Inherit `yaml:",inline"`
	// Cluster configuration
	Name                *string           `yaml:"name"`
	Namespace           *string           `yaml:"namespace"`
	DisableNamespace    *bool             `yaml:"disable-namespace"`
	APIDomain           *string           `yaml:"api-domain"`
	APIInsecureTLS      *bool             `yaml:"api-insecure-tls"`
	APIScheme           *string           `yaml:"api-scheme"`
	DebugAPIDomain      *string           `yaml:"debug-api-domain"`
	DebugAPIInsecureTLS *bool             `yaml:"debug-api-insecure-tls"`
	DebugAPIScheme      *string           `yaml:"debug-api-scheme"`
	NodeGroups          *ClusterNodeGroup `yaml:"node-groups"`
}

type ClusterNodeGroup map[string]struct {
	Mode      string `yaml:"mode"`
	BeeConfig string `yaml:"bee-config"`
	Config    string `yaml:"config"`
	Count     int    `yaml:"count"`
	Nodes     []struct {
		Name         string `yaml:"name"`
		Bootnodes    string `yaml:"bootnodes"`
		ClefKey      string `yaml:"clef-key"`
		ClefPassword string `yaml:"clef-password"`
		LibP2PKey    string `yaml:"libp2p-key"`
		SwarmKey     string `yaml:"swarm-key"`
	} `yaml:"nodes"`
}

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

func (c *Cluster) GetName() string {
	if c.Name == nil {
		return "noname"
	}
	return *c.Name
}

func (c *Cluster) GetNamespace() string {
	if c.Name == nil {
		return "nonamespace"
	}
	return *c.Namespace
}

func (c *Cluster) GetNodeGroups() ClusterNodeGroup {
	if c.NodeGroups == nil {
		return nil
	}
	return *c.NodeGroups
}

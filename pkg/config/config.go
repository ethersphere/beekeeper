package config

import (
	"fmt"
	"io/ioutil"
	"reflect"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Clusters map[string]struct {
		Name             string `yaml:"name"`
		Namespace        string `yaml:"namespace"`
		DisableNamespace bool   `yaml:"disable-namespace"`
		API              struct {
			Domain          string `yaml:"domain"`
			HostnamePattern string `yaml:"hostname-pattern"`
			InsecureTLS     bool   `yaml:"insecure-tls"`
			Scheme          string `yaml:"scheme"`
		} `yaml:"api"`
		DebugAPI struct {
			Domain          string `yaml:"domain"`
			HostnamePattern string `yaml:"hostname-pattern"`
			InsecureTLS     bool   `yaml:"insecure-tls"`
			Scheme          string `yaml:"scheme"`
		} `yaml:"debug-api"`
		NodeGroups map[string]struct {
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
		} `yaml:"node-groups"`
	} `yaml:"clusters"`
	BeeConfigs        map[string]BeeConfig        `yaml:"bee-configs"`
	CheckConfigs      map[string]CheckConfig      `yaml:"check-configs"`
	NodeGroupConfigs  map[string]NodeGroupConfig  `yaml:"node-group-configs"`
	SimulationConfigs map[string]SimulationConfig `yaml:"simulation-configs"`
	Kubernetes        struct {
		Enable     bool   `yaml:"enable"`
		InCluster  bool   `yaml:"in-cluster"`
		Kubeconfig string `yaml:"kubeconfig"`
	} `yaml:"kubernetes"`
}

type Inherit struct {
	ParrentName string `yaml:"_inherit"`
}

func (c *Config) Merge() (err error) {
	// merge BeeProfiles
	mergedBP := map[string]BeeConfig{}
	for name, v := range c.BeeConfigs {
		if len(v.ParrentName) == 0 {
			mergedBP[name] = v
		} else {
			parent, ok := c.BeeConfigs[v.ParrentName]
			if !ok {
				return fmt.Errorf("bee profile %s doesn't exist", v.ParrentName)
			}
			p := reflect.ValueOf(&parent).Elem()
			m := reflect.ValueOf(&v).Elem()
			for i := 0; i < m.NumField(); i++ {
				if m.Field(i).IsNil() && !p.Field(i).IsNil() {
					m.Field(i).Set(p.Field(i))
				}
			}
			mergedBP[name] = m.Interface().(BeeConfig)
		}
	}
	c.BeeConfigs = mergedBP

	// merge NodeGroupProfiles
	mergedNGP := map[string]NodeGroupConfig{}
	for name, v := range c.NodeGroupConfigs {
		if len(v.ParrentName) == 0 {
			mergedNGP[name] = v
		} else {
			parent, ok := c.NodeGroupConfigs[v.ParrentName]
			if !ok {
				return fmt.Errorf("node group profile %s doesn't exist", v.ParrentName)
			}
			p := reflect.ValueOf(&parent).Elem()
			m := reflect.ValueOf(&v).Elem()
			for i := 0; i < m.NumField(); i++ {
				if m.Field(i).IsNil() && !p.Field(i).IsNil() {
					m.Field(i).Set(p.Field(i))
				}
			}
			mergedNGP[name] = m.Interface().(NodeGroupConfig)
		}
	}
	c.NodeGroupConfigs = mergedNGP

	return
}

func Read(file string) (c *Config, err error) {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("yamlFile.Get err   %w ", err)
	}

	if err := yaml.Unmarshal(yamlFile, &c); err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}

	if err := c.Merge(); err != nil {
		return nil, fmt.Errorf("merging config: %w", err)
	}

	return
}

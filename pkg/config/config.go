package config

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"time"

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
	BeeProfiles       map[string]BeeProfile       `yaml:"bee-profiles"`
	NodeGroupProfiles map[string]NodeGroupProfile `yaml:"node-group-profiles"`
	CheckConfigs      map[string]CheckConfig      `yaml:"checks"`
	SimulationConfigs map[string]SimulationConfig `yaml:"simulations"`
	Kubernetes        struct {
		Enable     bool   `yaml:"enable"`
		InCluster  bool   `yaml:"in-cluster"`
		Kubeconfig string `yaml:"kubeconfig"`
	} `yaml:"kubernetes"`
}

type Profile struct {
	File    string `yaml:"_file"`
	Inherit string `yaml:"_inherit"`
}

type BeeProfile struct {
	Profile   `yaml:",inline"`
	BeeConfig `yaml:",inline"`
}

type NodeGroupProfile struct {
	Profile         `yaml:",inline"`
	NodeGroupConfig `yaml:",inline"`
}

type CheckConfig struct {
	Name    string         `yaml:"name"`
	Options yaml.Node      `yaml:"options"`
	Timeout *time.Duration `yaml:"timeout"`
}

type SimulationConfig struct {
	Name    string         `yaml:"name"`
	Options yaml.Node      `yaml:"options"`
	Timeout *time.Duration `yaml:"timeout"`
}

func (c *Config) Merge() (err error) {
	// merge BeeProfiles
	mergedBP := map[string]BeeProfile{}
	for name, v := range c.BeeProfiles {
		if len(v.Profile.Inherit) == 0 {
			mergedBP[name] = v
		} else {
			parent, ok := c.BeeProfiles[v.Profile.Inherit]
			if !ok {
				return fmt.Errorf("bee profile %s doesn't exist", v.Profile.Inherit)
			}
			p := reflect.ValueOf(&parent.BeeConfig).Elem()
			m := reflect.ValueOf(&v.BeeConfig).Elem()
			for i := 0; i < m.NumField(); i++ {
				if m.Field(i).IsNil() && !p.Field(i).IsNil() {
					m.Field(i).Set(p.Field(i))
				}
			}
			mergedBP[name] = BeeProfile{
				Profile:   v.Profile,
				BeeConfig: m.Interface().(BeeConfig),
			}
		}
	}
	c.BeeProfiles = mergedBP

	// merge NodeGroupProfiles
	mergedNGP := map[string]NodeGroupProfile{}
	for name, v := range c.NodeGroupProfiles {
		if len(v.Profile.Inherit) == 0 {
			mergedNGP[name] = v
		} else {
			parent, ok := c.NodeGroupProfiles[v.Profile.Inherit]
			if !ok {
				return fmt.Errorf("node group profile %s doesn't exist", v.Profile.Inherit)
			}
			p := reflect.ValueOf(&parent.NodeGroupConfig).Elem()
			m := reflect.ValueOf(&v.NodeGroupConfig).Elem()
			for i := 0; i < m.NumField(); i++ {
				if m.Field(i).IsNil() && !p.Field(i).IsNil() {
					m.Field(i).Set(p.Field(i))
				}
			}
			mergedNGP[name] = NodeGroupProfile{
				Profile:         v.Profile,
				NodeGroupConfig: m.Interface().(NodeGroupConfig),
			}
		}
	}
	c.NodeGroupProfiles = mergedNGP

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

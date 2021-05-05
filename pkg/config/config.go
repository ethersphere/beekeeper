package config

import (
	"fmt"
	"io/ioutil"
	"reflect"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Clusters    map[string]Cluster    `yaml:"clusters"`
	NodeGroups  map[string]NodeGroup  `yaml:"node-groups"`
	BeeConfigs  map[string]BeeConfig  `yaml:"bee-configs"`
	Checks      map[string]Check      `yaml:"checks"`
	Simulations map[string]Simulation `yaml:"simulations"`
}

type Inherit struct {
	ParrentName string `yaml:"_inherit"`
}

func (c *Config) Merge() (err error) {
	// merge BeeConfigs
	mergedBC := map[string]BeeConfig{}
	for name, v := range c.BeeConfigs {
		if len(v.ParrentName) == 0 {
			mergedBC[name] = v
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
			mergedBC[name] = m.Interface().(BeeConfig)
		}
	}
	c.BeeConfigs = mergedBC

	// merge NodeGroups
	mergedNG := map[string]NodeGroup{}
	for name, v := range c.NodeGroups {
		if len(v.ParrentName) == 0 {
			mergedNG[name] = v
		} else {
			parent, ok := c.NodeGroups[v.ParrentName]
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
			mergedNG[name] = m.Interface().(NodeGroup)
		}
	}
	c.NodeGroups = mergedNG

	// merge clusters
	mergedC := map[string]Cluster{}
	for name, v := range c.Clusters {
		if len(v.ParrentName) == 0 {
			mergedC[name] = v
		} else {
			parent, ok := c.Clusters[v.ParrentName]
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
			mergedC[name] = m.Interface().(Cluster)
		}
	}
	c.Clusters = mergedC

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

package config

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

// Config represents Beekeeper's configuration read from files
type Config struct {
	Clusters    map[string]Cluster    `yaml:"clusters"`
	NodeGroups  map[string]NodeGroup  `yaml:"node-groups"`
	BeeConfigs  map[string]BeeConfig  `yaml:"bee-configs"`
	Checks      map[string]Check      `yaml:"checks"`
	Simulations map[string]Simulation `yaml:"simulations"`
}

// Inherit is struct used for implementing inheritance in Config objects
type Inherit struct {
	ParentName string `yaml:"_inherit"`
}

// merge combines Config objects using inheritance
func (c *Config) merge() (err error) {
	// merge BeeConfigs
	mergedBC := map[string]BeeConfig{}
	for name, v := range c.BeeConfigs {
		if len(v.ParentName) == 0 {
			mergedBC[name] = v
		} else {
			parent, ok := c.BeeConfigs[v.ParentName]
			if !ok {
				return fmt.Errorf("bee profile %s doesn't exist", v.ParentName)
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
		if len(v.ParentName) == 0 {
			mergedNG[name] = v
		} else {
			parent, ok := c.NodeGroups[v.ParentName]
			if !ok {
				return fmt.Errorf("node group profile %s doesn't exist", v.ParentName)
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
		if len(v.ParentName) == 0 {
			mergedC[name] = v
		} else {
			parent, ok := c.Clusters[v.ParentName]
			if !ok {
				return fmt.Errorf("bee profile %s doesn't exist", v.ParentName)
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

// Read reads given YAML files and unmarshals them into Config
func Read(yamlFiles ...[]byte) (*Config, error) {
	c := Config{
		Clusters:    make(map[string]Cluster),
		NodeGroups:  make(map[string]NodeGroup),
		BeeConfigs:  make(map[string]BeeConfig),
		Checks:      make(map[string]Check),
		Simulations: make(map[string]Simulation),
	}

	for _, file := range yamlFiles {
		var tmp *Config
		if err := yaml.Unmarshal(file, &tmp); err != nil {
			return nil, fmt.Errorf("unmarshaling yaml file: %w", err)
		}

		// join Clusters
		for k, v := range tmp.Clusters {
			_, ok := c.Clusters[k]
			if !ok {
				c.Clusters[k] = v
			}
		}
		// join NodeGroups
		for k, v := range tmp.NodeGroups {
			_, ok := c.NodeGroups[k]
			if !ok {
				c.NodeGroups[k] = v
			}
		}
		// join BeeConfigs
		for k, v := range tmp.BeeConfigs {
			_, ok := c.BeeConfigs[k]
			if !ok {
				c.BeeConfigs[k] = v
			}
		}
		// join Checks
		for k, v := range tmp.Checks {
			_, ok := c.Checks[k]
			if !ok {
				c.Checks[k] = v
			}
		}
		// join Simulations
		for k, v := range tmp.Simulations {
			_, ok := c.Simulations[k]
			if !ok {
				c.Simulations[k] = v
			}
		}
	}

	// merge for inheritance
	if err := c.merge(); err != nil {
		return nil, fmt.Errorf("merging config: %w", err)
	}

	return &c, nil
}

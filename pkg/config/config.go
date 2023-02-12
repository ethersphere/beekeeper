package config

import (
	"fmt"
	"io"
	"reflect"

	"github.com/ethersphere/beekeeper/pkg/logging"
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

type YamlFile struct {
	Name    string
	Content []byte
}

// Inherit is struct used for implementing inheritance in Config objects
type Inherit struct {
	ParentName string `yaml:"_inherit"`
}

func (c *Config) PrintYaml(w io.Writer) (err error) {
	if c == nil {
		return fmt.Errorf("config not initialized")
	}
	enc := yaml.NewEncoder(w)
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("config can not be encoded: %s", err.Error())
	}
	return
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
func Read(log logging.Logger, yamlFiles []YamlFile) (*Config, error) {
	c := Config{
		Clusters:    make(map[string]Cluster),
		NodeGroups:  make(map[string]NodeGroup),
		BeeConfigs:  make(map[string]BeeConfig),
		Checks:      make(map[string]Check),
		Simulations: make(map[string]Simulation),
	}

	for _, file := range yamlFiles {
		var tmp *Config
		if err := yaml.Unmarshal(file.Content, &tmp); err != nil {
			return nil, fmt.Errorf("unmarshaling yaml file: %w", err)
		}

		// join Clusters
		for k, v := range tmp.Clusters {
			_, ok := c.Clusters[k]
			if !ok {
				c.Clusters[k] = v
			} else {
				log.Warningf("cluster '%s' in file '%s' already exits in configuration", k, file.Name)
			}
		}
		// join NodeGroups
		for k, v := range tmp.NodeGroups {
			_, ok := c.NodeGroups[k]
			if !ok {
				c.NodeGroups[k] = v
			} else {
				log.Warningf("node group '%s' in file '%s' already exits in configuration", k, file.Name)
			}
		}
		// join BeeConfigs
		for k, v := range tmp.BeeConfigs {
			_, ok := c.BeeConfigs[k]
			if !ok {
				c.BeeConfigs[k] = v
			} else {
				log.Warningf("bee config '%s' in file '%s' already exits in configuration", k, file.Name)
			}
		}
		// join Checks
		for k, v := range tmp.Checks {
			_, ok := c.Checks[k]
			if !ok {
				c.Checks[k] = v
			} else {
				log.Warningf("check '%s' in file '%s' already exits in configuration", k, file.Name)
			}
		}
		// join Simulations
		for k, v := range tmp.Simulations {
			_, ok := c.Simulations[k]
			if !ok {
				c.Simulations[k] = v
			} else {
				log.Warningf("simulation '%s' in file '%s' already exits in configuration", k, file.Name)
			}
		}
	}

	// merge for inheritance
	if err := c.merge(); err != nil {
		return nil, fmt.Errorf("merging config: %w", err)
	}

	return &c, nil
}

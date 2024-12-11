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
	if err := yaml.NewEncoder(w).Encode(c); err != nil {
		return fmt.Errorf("config can not be encoded: %s", err.Error())
	}
	return
}

// merge combines Config objects using inheritance
func (c *Config) merge() (err error) {
	c.BeeConfigs, err = mergeConfigs(c.BeeConfigs)
	if err != nil {
		return fmt.Errorf("merging bee configs: %w", err)
	}

	c.NodeGroups, err = mergeConfigs(c.NodeGroups)
	if err != nil {
		return fmt.Errorf("merging node groups: %w", err)
	}

	c.Clusters, err = mergeConfigs(c.Clusters)
	if err != nil {
		return fmt.Errorf("merging clusters: %w", err)
	}

	return
}

func mergeConfigs[T any](configs map[string]T) (map[string]T, error) {
	mergedConfigs := make(map[string]T)
	visited := map[string]bool{}

	// recursively merge configs (internal function)
	var mergeParent func(name string) (T, error)
	mergeParent = func(name string) (T, error) {
		if config, ok := mergedConfigs[name]; ok {
			return config, nil // already merged
		}
		var zero T

		// detect circular inheritance
		if visited[name] {
			return zero, fmt.Errorf("circular inheritance detected with bee profile %s", name)
		}
		visited[name] = true

		v, ok := configs[name]
		if !ok {
			return zero, fmt.Errorf("bee profile %s doesn't exist", name)
		}

		// check if T implements Inheritable to get parent name
		vIneheritable, ok := any(v).(Inheritable)
		if !ok {
			return zero, fmt.Errorf("type %T does not implement Inheritable interface", v)
		}

		// merge the parent
		if len(vIneheritable.GetParentName()) > 0 {
			parentConfig, err := mergeParent(vIneheritable.GetParentName())
			if err != nil {
				return zero, err
			}

			// merge parent fields into the current config
			p := reflect.ValueOf(&parentConfig).Elem()
			m := reflect.ValueOf(&v).Elem()
			for i := 0; i < m.NumField(); i++ {
				if m.Field(i).IsNil() && !p.Field(i).IsNil() {
					m.Field(i).Set(p.Field(i))
				}
			}
		}

		mergedConfigs[name] = v
		delete(visited, name) // remove after merge
		return v, nil
	}

	for name := range configs {
		if _, err := mergeParent(name); err != nil {
			return nil, err
		}
	}

	return mergedConfigs, nil
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
		log.Tracef("reading file %s", file.Name)

		var tmp *Config
		if err := yaml.Unmarshal(file.Content, &tmp); err != nil {
			return nil, fmt.Errorf("unmarshaling yaml file: %w", err)
		}

		// Set the cluster name to the key
		for key, cluster := range tmp.Clusters {
			cluster.Name = &key
			tmp.Clusters[key] = cluster
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

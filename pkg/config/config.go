package config

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Execute struct {
		Cluster  string `yaml:"cluster"`
		Playbook string `yaml:"playbook"`
	} `yaml:"execute"`
	// profiles
	BeeProfiles       map[string]BeeProfile       `yaml:"bee-profiles"`
	Clusters          map[string]Cluster          `yaml:"clusters"`
	Checks            map[string]CheckCfg         `yaml:"checks"`
	NodeGroupProfiles map[string]NodeGroupProfile `yaml:"node-group-profiles"`
	Playbooks         map[string]Playbook         `yaml:"playbooks"`
	// orchestrator
	Kubernetes struct {
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
	Profile `yaml:",inline"`
	Bee     `yaml:",inline"`
}

type NodeGroupProfile struct {
	Profile   `yaml:",inline"`
	NodeGroup `yaml:",inline"`
}

type Playbook struct {
	Checks              []string `yaml:"checks"`
	ChecksCommonOptions `yaml:",inline"`
	Stages              [][]struct {
		NodeGroup string `yaml:"node-group"`
		Add       int    `yaml:"add"`
		Start     int    `yaml:"start"`
		Stop      int    `yaml:"stop"`
		Delete    int    `yaml:"delete"`
	} `yaml:"stages"`
	Timeout time.Duration `yaml:"timeout"`
}

func (c *Config) Merge() (err error) {
	// TODO: generalize to have 1 function supporting all types
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
			p := reflect.ValueOf(&parent.Bee).Elem()
			m := reflect.ValueOf(&v.Bee).Elem()
			for i := 0; i < m.NumField(); i++ {
				if m.Field(i).IsNil() && !p.Field(i).IsNil() {
					m.Field(i).Set(p.Field(i))
				}
			}
			mergedBP[name] = BeeProfile{
				Profile: v.Profile,
				Bee:     m.Interface().(Bee),
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
			p := reflect.ValueOf(&parent.NodeGroup).Elem()
			m := reflect.ValueOf(&v.NodeGroup).Elem()
			for i := 0; i < m.NumField(); i++ {
				if m.Field(i).IsNil() && !p.Field(i).IsNil() {
					m.Field(i).Set(p.Field(i))
				}
			}
			mergedNGP[name] = NodeGroupProfile{
				Profile:   v.Profile,
				NodeGroup: m.Interface().(NodeGroup),
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

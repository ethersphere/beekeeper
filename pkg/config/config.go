package config

import (
	"io/ioutil"
	"log"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Cluster Cluster          `yaml:"cluster"` // TODO: add multi-cluster-support
	Run     map[string]Run   `yaml:"run"`
	Checks  map[string]Check `yaml:"checks"`
	// profiles
	BeeProfiles       map[string]BeeProfile       `yaml:"bee-profiles"`
	NodeGroupProfiles map[string]NodeGroupProfile `yaml:"node-group-profiles"`
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

type Run struct {
	Checks         []string `yaml:"checks"`
	MetricsEnabled bool     `yaml:"metrics-enabled"`
	Seed           int64    `yaml:"seed"`
	Stages         [][]struct {
		NodeGroup string `yaml:"node-group"`
		Add       int    `yaml:"add"`
		Start     int    `yaml:"start"`
		Stop      int    `yaml:"stop"`
		Delete    int    `yaml:"delete"`
	} `yaml:"stages"`
	Timeout time.Duration `yaml:"timeout"`
}

type BeeProfile struct {
	Profile `yaml:",inline"`
	Bee     `yaml:",inline"`
}

type NodeGroupProfile struct {
	Profile   `yaml:",inline"`
	NodeGroup `yaml:",inline"`
}

func (c *Config) Merge() {
	// TODO: generalize to have 1 function supporting all types
	// merge BeeProfiles
	mergedBP := map[string]BeeProfile{}
	for name, v := range c.BeeProfiles {
		if len(v.Profile.Inherit) == 0 {
			mergedBP[name] = v
		} else {
			parent := c.BeeProfiles[v.Profile.Inherit]
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
			parent := c.NodeGroupProfiles[v.Profile.Inherit]
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
}

func Read(file string) (c *Config) {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	if err := yaml.Unmarshal(yamlFile, &c); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	c.Merge()

	return
}

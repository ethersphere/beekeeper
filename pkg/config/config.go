package config

import (
	"io/ioutil"
	"log"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Cluster    Cluster `yaml:"cluster"`
	Kubernetes struct {
		Enable     bool   `yaml:"enable"`
		InCluster  bool   `yaml:"in-cluster"`
		Kubeconfig string `yaml:"kubeconfig"`
	} `yaml:"kubernetes"`
	Run Run `yaml:"run"`
	// profiles
	BeeProfiles       map[string]BeeProfile       `yaml:"bee-profiles"`
	CheckProfiles     map[string]CheckProfile     `yaml:"check-profiles"`
	NodeGroupProfiles map[string]NodeGroupProfile `yaml:"node-group-profiles"`
}

type Profile struct {
	File    string `yaml:"_file"`
	Inherit string `yaml:"_inherit"`
}

type Run struct {
	Checks  []string      `yaml:"checks"`
	Seed    int64         `yaml:"seed"`
	Timeout time.Duration `yaml:"timeout"`
}

type BeeProfile struct {
	Profile `yaml:",inline"`
	Bee     `yaml:",inline"`
}

type CheckProfile struct {
	Profile `yaml:",inline"`
	Check   `yaml:",inline"`
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

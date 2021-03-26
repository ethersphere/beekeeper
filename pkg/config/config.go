package config

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Cluster           Cluster               `yaml:"cluster"`
	Check             Check                 `yaml:"check"`
	BeeProfiles       map[string]BeeProfile `yaml:"bee-profiles"`
	NodeGroupProfiles map[string]NodeGroup  `yaml:"node-group-profiles"`
	Kubernetes        struct {
		Kubeconfig string `yaml:"kubeconfig"`
		InCluster  bool   `yaml:"in-cluster"`
	} `yaml:"kubernetes"`
}

type Profile struct {
	File    string `yaml:"_file"`
	Inherit string `yaml:"_inherit"`
}

func (c *Config) Merge() {
	// merge BeeProfiles
	for name, v := range c.BeeProfiles {
		if len(v.Profile.Inherit) > 0 {
			fmt.Println(name, "from", v.Profile.Inherit)
		}
	}

	// merge NodeGroupProfiles
	for name, v := range c.NodeGroupProfiles {
		if len(v.Profile.Inherit) > 0 {
			fmt.Println(name, "from", v.Profile.Inherit)
		}
	}
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

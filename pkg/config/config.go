package config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Cluster struct {
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
		NodeGroups []struct {
			Name      string `yaml:"name"`
			Mode      string `yaml:"mode"`
			BeeConfig string `yaml:"bee-config"`
			Config    string `yaml:"config"`
			Count     string `yaml:"count"`
			Nodes     []struct {
				Bootnodes    string `yaml:"bootnodes"`
				ClefKey      string `yaml:"clef-key"`
				ClefPassword string `yaml:"clef-password"`
				LibP2PKey    string `yaml:"libp2p-key"`
				SwarmKey     string `yaml:"swarm-key"`
			} `yaml:"nodes"`
		} `yaml:"node-groups"`
	} `yaml:"cluster"`
	Kubernetes struct {
		Kubeconfig string `yaml:"kubeconfig"`
		InCluster  bool   `yaml:"in-cluster"`
	} `yaml:"kubernetes"`
}

func (c *Config) Read() *Config {
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

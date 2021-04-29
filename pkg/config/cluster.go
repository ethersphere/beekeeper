package config

type Cluster struct {
	Name                    string `yaml:"name"`
	Namespace               string `yaml:"namespace"`
	DisableNamespace        bool   `yaml:"disable-namespace"`
	APIDomain               string `yaml:"api-domain"`
	APIHostnamePattern      string `yaml:"api-hostname-pattern"`
	APIInsecureTLS          bool   `yaml:"api-insecure-tls"`
	APIScheme               string `yaml:"api-scheme"`
	DebugAPIDomain          string `yaml:"debug-api-domain"`
	DebugAPIHostnamePattern string `yaml:"debug-api-hostname-pattern"`
	DebugAPIInsecureTLS     bool   `yaml:"debug-api-insecure-tls"`
	DebugAPIScheme          string `yaml:"debug-api-scheme"`
	NodeGroups              map[string]struct {
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
}

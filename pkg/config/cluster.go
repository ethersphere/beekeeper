package config

type Cluster struct {
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
}

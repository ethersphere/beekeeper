package config

type Check struct {
	Seed string `yaml:"seed"`
	Run  []struct {
		Name    string `yaml:"name"`
		Initial []struct {
			Name  string `yaml:"name"`
			Count int    `yaml:"count"`
		} `yaml:"initial"`
		Stages [][]struct {
			NodeGroup string `yaml:"node-group"`
			Add       int    `yaml:"add"`
			Start     int    `yaml:"start"`
			Stop      int    `yaml:"stop"`
			Delete    int    `yaml:"delete"`
		} `yaml:"stages"`
	} `yaml:"run"`
}

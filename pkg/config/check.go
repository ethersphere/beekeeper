package config

import "time"

type Check struct {
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
	Timeout *time.Duration `yaml:"timeout"`
}

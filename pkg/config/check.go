package config

import (
	"time"

	"gopkg.in/yaml.v3"
)

type Check struct {
	Name string `yaml:"name"`
	// Initial []struct {
	// 	Name  string `yaml:"name"`
	// 	Count int    `yaml:"count"`
	// } `yaml:"initial"`
	Options yaml.Node `yaml:"options"`
	// Stages  [][]struct {
	// 	NodeGroup string `yaml:"node-group"`
	// 	Add       int    `yaml:"add"`
	// 	Start     int    `yaml:"start"`
	// 	Stop      int    `yaml:"stop"`
	// 	Delete    int    `yaml:"delete"`
	// } `yaml:"stages"`
	Timeout *time.Duration `yaml:"timeout"`
}

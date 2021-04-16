package config

import (
	"time"

	"gopkg.in/yaml.v3"
)

type Check struct {
	Name    string         `yaml:"name"`
	Options yaml.Node      `yaml:"options"`
	Timeout *time.Duration `yaml:"timeout"`
}

type ChecksCommonOptions struct {
	MetricsEnabled bool  `yaml:"metrics-enabled"`
	Seed           int64 `yaml:"seed"`
}

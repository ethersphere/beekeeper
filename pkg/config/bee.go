package config

import (
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

type Inheritable interface {
	GetParentName() string
}

// BeeConfig represents Bee configuration as read from Beekeeper's YAML config.
//
// It embeds orchestration.Config (the Bee flag set), so the flag fields are
// defined in exactly one place and the same yaml tags — the Bee flag names —
// are used both for reading the config here and for rendering the node's
// .bee.yaml. The only thing BeeConfig adds is the config-file-only concern of
// inheritance (_inherit). Export returns just the embedded Config, so neither
// _inherit nor any other config-loading detail can leak into the rendered file.
type BeeConfig struct {
	*Inherit             `yaml:",inline"`
	orchestration.Config `yaml:",inline"`
}

func (b BeeConfig) GetParentName() string {
	if b.Inherit != nil {
		return b.ParentName
	}
	return ""
}

// Export returns the Bee flag configuration to be rendered into the node's
// .bee.yaml. Inheritance has already been resolved during config loading and is
// not part of the embedded Config.
func (b *BeeConfig) Export() orchestration.Config {
	return b.Config
}

package config

import (
	"github.com/bluemedora/bplogagent/plugin"
)

type Config struct {
	Plugins    []plugin.PluginConfig
	BundlePath string `mapstructure:"bundle_path" yaml:"bundle_path"`
}

func (c Config) IsZero() bool {
	return len(c.Plugins) == 0 && c.BundlePath == ""
}

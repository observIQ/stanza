package config

import (
	"github.com/bluemedora/bplogagent/plugin"
)

type Config struct {
	Plugins      []plugin.PluginConfig
	BundlePath   string `mapstructure:"bundle_path" yaml:"bundle_path"`
	DatabaseFile string `mapstructure:"database_file" yaml:"database_file"`
}

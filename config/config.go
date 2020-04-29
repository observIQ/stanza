package config

import (
	"github.com/bluemedora/bplogagent/plugin"
)

type Config struct {
	Plugins           []plugin.Config `mapstructure:"plugins"`
	PluginGraphOutput string          `mapstructure:"graph"`
	BundleDir         string          `mapstructure:"bundle_dir" yaml:"bundle_dir,omitempty"`
	DatabaseFile      string          `mapstructure:"database_file" yaml:"database_file,omitempty"`
}

func (c Config) IsZero() bool {
	return len(c.Plugins) == 0 && c.BundleDir == "" && c.PluginGraphOutput == "" && c.DatabaseFile == ""
}

var DecodeHookFunc = plugin.ConfigDecoder

package config

import (
	"github.com/bluemedora/bplogagent/plugin"
)

type Config struct {
	Plugins           []plugin.Config `mapstructure:"plugins"       json:"plugins"                 yaml:"plugins"`
	PluginGraphOutput string          `mapstructure:"graph"         json:"graph,omitempty"         yaml:"graph,omitempty"`
	DatabaseFile      string          `mapstructure:"database_file" json:"database_file,omitempty" yaml:"database_file,omitempty"`
}

func (c Config) IsZero() bool {
	return len(c.Plugins) == 0 && c.PluginGraphOutput == "" && c.DatabaseFile == ""
}

var DecodeHookFunc = plugin.ConfigDecoder

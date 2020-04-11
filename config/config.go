package config

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	Plugins      []plugin.Config
	BundlePath   string `mapstructure:"bundle_path" yaml:"bundle_path,omitempty"`
	DatabaseFile string `mapstructure:"database_file" yaml:"database_file,omitempty"`
}

func (c Config) IsZero() bool {
	return len(c.Plugins) == 0 && c.BundlePath == ""
}

var DecodeHookFunc = mapstructure.ComposeDecodeHookFunc(plugin.ConfigDecoder, entry.FieldSelectorDecoder)

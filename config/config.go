package config

import (
	"github.com/bluemedora/bplogagent/plugin"
)

type Config struct {
	// TODO make a PluginConfigList type that can validate that the outputs exist
	// when parsing the config? Also can define a .Build() method on it to get
	// a PluginList
	Plugins []plugin.PluginConfig
}

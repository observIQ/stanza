package config

import "github.com/bluemedora/log-agent/plugin"

type Config struct {
	LogLevel string
	LogPath  string
	Plugins  []plugin.PluginConfig `mapstructure:"plugins"`
}

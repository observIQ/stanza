package config

import "github.com/BlueMedora/log-agent/plugin"

type Config struct {
	LogLevel string
	LogPath  string
	Plugins  []plugin.PluginConfig
}

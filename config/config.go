package config

import (
	"github.com/mitchellh/mapstructure"
	"go.bluemedora.com/bplogagent/plugin"
)

type Config struct {
	Plugins []plugin.PluginConfig
}

func UnmarshalHook(c *mapstructure.DecoderConfig) {
	c.DecodeHook = plugin.PluginConfigToStructHookFunc()
}

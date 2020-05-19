package helper

import (
	"github.com/bluemedora/bplogagent/errors"
	"go.uber.org/zap"
)

// BasicPluginConfig provides a basic implemention for a plugin config.
type BasicPluginConfig struct {
	PluginID   string `mapstructure:"id"   json:"id"   yaml:"id"`
	PluginType string `mapstructure:"type" json:"type" yaml:"type"`
}

// ID will return the plugin id.
func (c BasicPluginConfig) ID() string {
	return c.PluginID
}

// Type will return the plugin type.
func (c BasicPluginConfig) Type() string {
	return c.PluginType
}

// Build will build a basic plugin.
func (c BasicPluginConfig) Build(logger *zap.SugaredLogger) (BasicPlugin, error) {
	if c.PluginID == "" {
		return BasicPlugin{}, errors.NewError(
			"Plugin config is missing the `id` field.",
			"Ensure that all plugins have a uniquely defined `id` field.",
		)
	}

	if c.PluginType == "" {
		return BasicPlugin{}, errors.NewError(
			"Plugin config is missing the `type` field.",
			"",
			"plugin_id", c.PluginID,
		)
	}

	plugin := BasicPlugin{
		PluginID:      c.PluginID,
		PluginType:    c.PluginType,
		SugaredLogger: logger.With("plugin_id", c.PluginID, "plugin_type", c.PluginType),
	}

	return plugin, nil
}

// BasicPlugin provides a basic implementation of a plugin.
type BasicPlugin struct {
	PluginID   string
	PluginType string
	*zap.SugaredLogger
}

// ID will return the plugin id.
func (b *BasicPlugin) ID() string {
	return b.PluginID
}

// Type will return the plugin type.
func (b *BasicPlugin) Type() string {
	return b.PluginType
}

// Logger returns the plugin's scoped logger to satisfy plugin.Plugin
func (b *BasicPlugin) Logger() *zap.SugaredLogger {
	return b.SugaredLogger
}

package helper

import (
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap"
)

// BasicConfig provides a basic implemention for a plugin config.
type BasicConfig struct {
	PluginID   string `mapstructure:"id"   json:"id"   yaml:"id"`
	PluginType string `mapstructure:"type" json:"type" yaml:"type"`
}

// ID will return the plugin id.
func (c BasicConfig) ID() string {
	return c.PluginID
}

// Type will return the plugin type.
func (c BasicConfig) Type() string {
	return c.PluginType
}

// Build will build a basic plugin.
func (c BasicConfig) Build(context plugin.BuildContext) (BasicPlugin, error) {
	if c.PluginID == "" {
		return BasicPlugin{}, errors.NewError(
			"Plugin config is missing the `id` field.",
			"Ensure that all plugins have a uniquely defined `id` field.",
		)
	}

	if c.PluginType == "" {
		return BasicPlugin{}, errors.NewError(
			"Plugin config is missing the `type` field.",
			"Ensure that all plugins have a uniquely defined `type` field.",
			"plugin_id", c.PluginID,
		)
	}

	plugin := BasicPlugin{
		PluginID:      c.PluginID,
		PluginType:    c.PluginType,
		SugaredLogger: context.Logger.With("plugin_id", c.PluginID, "plugin_type", c.PluginType),
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
func (p *BasicPlugin) ID() string {
	return p.PluginID
}

// Type will return the plugin type.
func (p *BasicPlugin) Type() string {
	return p.PluginType
}

// Logger returns the plugin's scoped logger.
func (p *BasicPlugin) Logger() *zap.SugaredLogger {
	return p.SugaredLogger
}

// Start will start the plugin.
func (p *BasicPlugin) Start() error {
	return nil
}

// Stop will stop the plugin.
func (p *BasicPlugin) Stop() error {
	return nil
}

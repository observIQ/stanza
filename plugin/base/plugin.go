package base

import (
	"fmt"

	"github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap"
)

// PluginConfig defines how to configure and build a basic plugin.
type PluginConfig struct {
	PluginID   string `mapstructure:"id" yaml:"id"`
	PluginType string `mapstructure:"type" yaml:"type"`
}

// ID returns the id field from the plugin config.
func (c *PluginConfig) ID() string {
	return c.PluginID
}

// Type returns the type field from the plugin config.
func (c *PluginConfig) Type() string {
	return c.PluginType
}

// Build will build a basic plugin.
func (c *PluginConfig) Build(context plugin.BuildContext) (Plugin, error) {
	if c.PluginID == "" {
		return Plugin{}, fmt.Errorf("missing required field 'id'")
	}

	if c.PluginType == "" {
		return Plugin{}, fmt.Errorf("missing required field 'type'")
	}

	plugin := Plugin{
		PluginID:      c.PluginID,
		PluginType:    c.PluginType,
		SugaredLogger: context.Logger.With("plugin_id", c.PluginID, "plugin_type", c.PluginType),
	}

	return plugin, nil
}

// Plugin satisfies the basic requirements of a plugin.
type Plugin struct {
	PluginID   string
	PluginType string
	*zap.SugaredLogger
}

// ID will return the plugin id of the plugin.
func (p *Plugin) ID() string {
	return p.PluginID
}

// Type will return the plugin type of the plugin.
func (p *Plugin) Type() string {
	return p.PluginType
}

// Start will log that the plugin has started.
func (p *Plugin) Start() error {
	p.Debug("Ignoring startup. Not implemented.")
	return nil
}

// Stop will log that the plugin has stopped.
func (p *Plugin) Stop() error {
	p.Debug("Ignoring shutdown. Not implemented.")
	return nil
}

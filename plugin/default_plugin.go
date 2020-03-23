package plugin

import (
	"fmt"

	"go.uber.org/zap"
)

type DefaultPluginConfig struct {
	PluginID   PluginID `mapstructure:"id" yaml:"id"`
	PluginType string   `mapstructure:"type" yaml:"type"`
}

func (c DefaultPluginConfig) Build(logger *zap.SugaredLogger) (DefaultPlugin, error) {
	if c.PluginID == "" {
		return DefaultPlugin{}, fmt.Errorf("missing required field 'id'")
	}

	if c.Type() == "" {
		return DefaultPlugin{}, fmt.Errorf("missing required field 'type'")
	}

	plugin := DefaultPlugin{
		PluginID:      c.PluginID,
		PluginType:    c.Type(),
		SugaredLogger: logger.With("plugin_id", c.PluginID),
	}

	return plugin, nil
}

func (c DefaultPluginConfig) ID() PluginID {
	return c.PluginID
}

func (c DefaultPluginConfig) Type() string {
	return c.PluginType
}

type DefaultPlugin struct {
	PluginID   PluginID
	PluginType string
	*zap.SugaredLogger
}

func (p *DefaultPlugin) ID() PluginID {
	return p.PluginID
}

func (p *DefaultPlugin) Type() string {
	return p.PluginType
}

func (p *DefaultPlugin) Start() error {
	p.Debug("Plugin started")
	return nil
}

func (p *DefaultPlugin) Stop() error {
	p.Debug("Plugin stopped")
	return nil
}

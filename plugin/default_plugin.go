package plugin

import "fmt"

type DefaultPluginConfig struct {
	PluginID   PluginID `mapstructure:"id"`
	PluginType string   `mapstructure:"type"`
}

func (c DefaultPluginConfig) Build() (DefaultPlugin, error) {
	if c.PluginID == "" {
		return DefaultPlugin{}, fmt.Errorf("missing required field 'id'")
	}

	if c.Type() == "" {
		return DefaultPlugin{}, fmt.Errorf("missing required field 'type'")
	}

	plugin := DefaultPlugin{
		id:         c.PluginID,
		pluginType: c.Type(),
	}

	return plugin, nil
}

func (c DefaultPluginConfig) ID() PluginID {
	return c.PluginID
}

func (c DefaultPluginConfig) Type() string {
	return c.Type()
}

type DefaultPlugin struct {
	id         PluginID
	pluginType string
}

func (p *DefaultPlugin) ID() PluginID {
	return p.id
}

func (p *DefaultPlugin) Type() string {
	return p.pluginType
}

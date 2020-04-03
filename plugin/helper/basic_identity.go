package helper

import (
	"fmt"

	"go.uber.org/zap"
)

// BasicIdentityConfig provides a basic implemention for plugin config identity.
type BasicIdentityConfig struct {
	PluginID   string `mapstructure:"id" yaml:"id"`
	PluginType string `mapstructure:"type" yaml:"type"`
}

// ID will return the plugin id.
func (c BasicIdentityConfig) ID() string {
	return c.PluginID
}

// Type will return the plugin type.
func (c BasicIdentityConfig) Type() string {
	return c.PluginType
}

// Build will build a basic identity.
func (c BasicIdentityConfig) Build(logger *zap.SugaredLogger) (BasicIdentity, error) {
	if c.PluginID == "" {
		return BasicIdentity{}, fmt.Errorf("missing field 'id'")
	}

	if c.PluginType == "" {
		return BasicIdentity{}, fmt.Errorf("missing field 'type'")
	}

	plugin := BasicIdentity{
		PluginID:      c.PluginID,
		PluginType:    c.PluginType,
		SugaredLogger: logger.With("plugin_id", c.PluginID, "plugin_type", c.PluginType),
	}

	return plugin, nil
}

// BasicIdentity provides a basic implementation of plugin identity.
type BasicIdentity struct {
	PluginID   string
	PluginType string
	*zap.SugaredLogger
}

// ID will return the plugin id.
func (b *BasicIdentity) ID() string {
	return b.PluginID
}

// Type will return the plugin type.
func (b *BasicIdentity) Type() string {
	return b.PluginType
}

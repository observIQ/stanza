package base

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

// OutputConfig defines how to configure and build a basic output plugin.
type OutputConfig struct {
	PluginConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a basic output plugin.
func (c OutputConfig) Build(context plugin.BuildContext) (OutputPlugin, error) {
	p, err := c.PluginConfig.Build(context)
	if err != nil {
		return OutputPlugin{}, err
	}

	return OutputPlugin{p}, nil
}

// OutputPlugin is a plugin that is a consumer, but not a producer.
type OutputPlugin struct {
	Plugin
}

// Consume will log that an entry has been consumed.
func (o *OutputPlugin) Consume(entry *entry.Entry) error {
	o.Debug("Entry consumed")
	return nil
}

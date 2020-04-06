package builtin

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("noop", &NoopPluginConfig{})
}

// NoopPluginConfig is the configuration of a noop plugin.
type NoopPluginConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`
}

// Build will build a noop plugin.
func (c NoopPluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicTransformer, err := c.BasicTransformerConfig.Build()
	if err != nil {
		return nil, err
	}

	noopPlugin := &NoopPlugin{
		BasicPlugin:      basicPlugin,
		BasicTransformer: basicTransformer,
	}

	return noopPlugin, nil
}

// NoopPlugin is a plugin that performs no operations on an entry.
type NoopPlugin struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicTransformer
}

// Process will forward the entry to the next output without any alterations.
func (p *NoopPlugin) Process(entry *entry.Entry) error {
	return p.Output.Process(entry)
}

package builtin

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("noop", &NoopPluginConfig{})
}

// NoopPluginConfig is the configuration of a noop plugin.
type NoopPluginConfig struct {
	helper.TransformerConfig `yaml:",inline"`
}

// Build will build a noop plugin.
func (c NoopPluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	noopPlugin := &NoopPlugin{
		TransformerPlugin: transformerPlugin,
	}

	return noopPlugin, nil
}

// NoopPlugin is a plugin that performs no operations on an entry.
type NoopPlugin struct {
	helper.TransformerPlugin
}

// Process will forward the entry to the next output without any alterations.
func (p *NoopPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	return p.Output.Process(ctx, entry)
}

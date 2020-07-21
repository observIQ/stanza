package transformer

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("noop", func() plugin.Builder { return NewNoopPluginConfig("") })
}

func NewNoopPluginConfig(pluginID string) *NoopPluginConfig {
	return &NoopPluginConfig{
		TransformerConfig: helper.NewTransformerConfig(pluginID, "noop"),
	}
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
	p.Write(ctx, entry)
	return nil
}

package base

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

// FilterConfig defines how to configure and build a basic filter plugin.
type FilterConfig struct {
	PluginConfig `mapstructure:",squash" yaml:",inline"`
	OutputID     string `mapstructure:"output" yaml:"output"`
}

// Build will build a basic filter plugin.
func (c *FilterConfig) Build(context plugin.BuildContext) (FilterPlugin, error) {
	p, err := c.PluginConfig.Build(context)
	if err != nil {
		return FilterPlugin{}, err
	}

	if c.OutputID == "" {
		return FilterPlugin{}, fmt.Errorf("output parameter is not defined")
	}

	filterPlugin := FilterPlugin{
		Plugin:   p,
		OutputID: c.OutputID,
	}

	return filterPlugin, nil
}

// FilterPlugin is a plugin that is plugin that filters log traffic.
type FilterPlugin struct {
	Plugin
	OutputID string
	Output   plugin.Consumer
}

// Consumers will return an array containing the plugin's connected output.
func (p *FilterPlugin) Consumers() []plugin.Consumer {
	return []plugin.Consumer{p.Output}
}

// SetConsumers will find and set the consumer that matches the output id.
func (p *FilterPlugin) SetConsumers(consumers []plugin.Consumer) error {
	consumer, err := FindConsumer(consumers, p.OutputID)
	if err != nil {
		return err
	}

	p.Output = consumer
	return nil
}

// Consume will log that an entry has been filtered.
func (p *FilterPlugin) Consume(entry *entry.Entry) error {
	return fmt.Errorf("consume not implemented")
}

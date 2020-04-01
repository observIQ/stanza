package base

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

// ParserConfig defines how to configure and build a basic parser plugin.
type ParserConfig struct {
	PluginConfig `mapstructure:",squash" yaml:",inline"`
	OutputID     string `mapstructure:"output" yaml:"output"`
}

// Build will build a basic parser plugin.
func (c *ParserConfig) Build(context plugin.BuildContext) (ParserPlugin, error) {
	p, err := c.PluginConfig.Build(context)
	if err != nil {
		return ParserPlugin{}, err
	}

	if c.OutputID == "" {
		return ParserPlugin{}, fmt.Errorf("output parameter is not defined")
	}

	parserPlugin := ParserPlugin{
		Plugin: p,
		OutputID: c.OutputID,
	}

	return parserPlugin, nil
}

// ParserPlugin is a plugin that parses a field in an entry.
type ParserPlugin struct {
	Plugin
	OutputID string
	Output   plugin.Consumer
}

// Consumers will return an array containing the plugin's connected output.
func (p *ParserPlugin) Consumers() []plugin.Consumer {
	return []plugin.Consumer{p.Output}
}

// SetConsumers will find and set the consumer that matches the output id.
func (p *ParserPlugin) SetConsumers(consumers []plugin.Consumer) error {
	consumer, err := FindConsumer(consumers, p.OutputID)
	if err != nil {
		return err
	}

	p.Output = consumer
	return nil
}

// Consume will log that an entry has been parsed.
func (p *ParserPlugin) Consume(e *entry.Entry) error {
	return fmt.Errorf("consume not implemented")
}

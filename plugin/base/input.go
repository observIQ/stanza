package base

import (
	"fmt"

	"github.com/bluemedora/bplogagent/plugin"
)

// InputConfig defines how to configure and build a basic input plugin.
type InputConfig struct {
	PluginConfig `mapstructure:",squash" yaml:",inline"`
	OutputID     string `mapstructure:"output" yaml:"output"`
}

// Build will build a basic input plugin.
func (c *InputConfig) Build(context plugin.BuildContext) (InputPlugin, error) {
	p, err := c.PluginConfig.Build(context)
	if err != nil {
		return InputPlugin{}, err
	}

	if c.OutputID == "" {
		return InputPlugin{}, fmt.Errorf("output parameter is not defined")
	}

	inputPlugin := InputPlugin{
		Plugin:   p,
		OutputID: c.OutputID,
	}

	return inputPlugin, nil
}

// InputPlugin is a plugin that is a producer, but not a consumer.
type InputPlugin struct {
	Plugin
	OutputID string
	Output   plugin.Consumer
}

// Consumers will return an array containing the plugin's connected output.
func (p *InputPlugin) Consumers() []plugin.Consumer {
	return []plugin.Consumer{p.Output}
}

// SetConsumers will find and set the consumer that matches the output id.
func (p *InputPlugin) SetConsumers(consumers []plugin.Consumer) error {
	consumer, err := FindConsumer(consumers, p.OutputID)
	if err != nil {
		return err
	}

	p.Output = consumer
	return nil
}

// FindConsumer will find a consumer with the supplied id.
func FindConsumer(consumers []plugin.Consumer, consumerID string) (plugin.Consumer, error) {
	for _, consumer := range consumers {
		if consumer.ID() == consumerID {
			return consumer, nil
		}
	}
	return nil, fmt.Errorf("consumer %s is missing", consumerID)
}
